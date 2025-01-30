package api

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/hyle-team/tss-svc/internal/api/common"
	"github.com/hyle-team/tss-svc/internal/api/ctx"
	"github.com/hyle-team/tss-svc/internal/api/requests"
	apiTypes "github.com/hyle-team/tss-svc/internal/api/types"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/clients"
	database "github.com/hyle-team/tss-svc/internal/db"
	types "github.com/hyle-team/tss-svc/internal/types"
	"gitlab.com/distributed_lab/ape"
	"gitlab.com/distributed_lab/ape/problems"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	paramChainId = "chain_id"
	paramTxHash  = "tx_hash"
	paramTxNonce = "tx_nonce"

	pollingPeriod = 1 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func CheckWithdrawalWs(w http.ResponseWriter, r *http.Request) {
	var (
		ctxt    = r.Context()
		clients = ctx.Clients(ctxt)
	)

	//get incoming params
	depositIdentifier, err := parseIncomingUrlParams(r, clients)
	if err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		return
	}

	_, err = requests.CheckTx(ctxt, &types.DepositIdentifier{
		TxHash:  depositIdentifier.TxHash,
		TxNonce: uint32(depositIdentifier.TxNonce),
		ChainId: depositIdentifier.ChainId,
	})
	if err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		ctx.Logger(ctxt).WithError(err).Debug("websocket upgrade error")
		return
	}

	gracefulClose := make(chan struct{})
	go watchConnectionClosing(ws, gracefulClose)
	watchWithdrawalStatus(ctxt, ws, gracefulClose, *depositIdentifier)
}

func watchConnectionClosing(ws *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		// collecting errors and close message to signalize writer.
		// note: `ReadMessage` is a blocking operation.
		// note: infinite loop will be broken either by close message or
		//       closed connection by writer goroutine, which immediately
		//       sends an error to a reader.
		mt, _, err := ws.ReadMessage()
		if err != nil || mt == websocket.CloseMessage {
			break
		}
	}
}

func watchWithdrawalStatus(ctxt context.Context, ws *websocket.Conn, connClosed chan struct{}, id database.DepositIdentifier) {
	defer func() { _ = ws.Close() }()

	var (
		db                                         = ctx.DB(ctxt)
		logger                                     = ctx.Logger(ctxt)
		prevStatus          types.WithdrawalStatus = -1
		cancelled, graceful bool
		ticker              = time.NewTicker(pollingPeriod)

		// function to repeat iteration after some period or break the loop
		// in case of a cancellation signal. If the signal is produced by
		// app context, websocket connection would be closed gracefully with
		// the corresponding `CloseGoingAway` status
		tillCancel = func() {
			select {
			case <-connClosed:
				cancelled = true
			case <-ctxt.Done():
				cancelled, graceful = true, true
			case <-ticker.C:
				// doing nothing, just waiting some period
			}
		}
	)

	defer ticker.Stop()

	// fast-starting without waiting for initial tick.
	// This shenanigan is just a classic `do-while` construction
	// with missing init statement and condition expression.
	// Using `tillCancel` as a post statement allows us to run
	// first iteration without waiting for ticker to tick.
	for ; ; tillCancel() {
		if cancelled {
			if graceful {
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"))
			}
			return
		}

		deposit, err := db.Get(id)
		if err != nil {
			logger.WithError(err).Error("failed to get withdrawal")
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Internal server error"))
			return
		}
		if deposit == nil {
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4004, "withdrawal not found"))
			return
		}

		//poll until our tx won`t be finished
		if deposit.WithdrawalStatus == prevStatus {
			continue
		}

		raw, err := protojson.Marshal(common.ToStatusResponse(deposit))
		if err != nil {
			logger.WithError(err).Error("failed to marshal withdrawal status")
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Internal server error"))
			return
		}
		if err = ws.WriteMessage(websocket.TextMessage, raw); err != nil {
			logger.WithError(err).Error("failed to write message to websocket")
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Internal server error"))
			return
		}

		// is it a time for websocket closing
		if slices.Contains(database.FinalWithdrawalStatuses, deposit.WithdrawalStatus) {
			err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logger.WithError(err).Error("failed to send close msg after finish")
			}
			return
		}

		prevStatus = deposit.WithdrawalStatus
	}
}

func parseIncomingUrlParams(r *http.Request, clients bridgeTypes.ClientsRepository) (*database.DepositIdentifier, error) {
	chainId := chi.URLParam(r, paramChainId)
	if _, err := clients.Client(chainId); err != nil {
		return nil, apiTypes.ErrInvalidChainId
	}
	txHash := chi.URLParam(r, paramTxHash)
	if len(txHash) < 3 {

		return nil, apiTypes.ErrInvalidTxHash
	}
	txNonce, err := strconv.Atoi(chi.URLParam(r, paramTxNonce))
	if err != nil || txNonce < 0 {

		return nil, apiTypes.ErrInvalidTxNonce
	}

	return &database.DepositIdentifier{
		TxHash:  txHash,
		TxNonce: txNonce,
		ChainId: chainId,
	}, nil
}

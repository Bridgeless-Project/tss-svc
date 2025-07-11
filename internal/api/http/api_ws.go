package http

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/api/common"
	"github.com/Bridgeless-Project/tss-svc/internal/api/ctx"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/ape"
	"gitlab.com/distributed_lab/ape/problems"
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
		ctxt   = r.Context()
		logger = ctx.Logger(ctxt)
		db     = ctx.DB(ctxt)
	)

	depositIdentifier, err := identifierFromParams(r)
	if err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		return
	}

	deposit, err := db.Get(*depositIdentifier)
	if err != nil {
		logger.WithError(err).Error("failed to get withdrawal")
		ape.RenderErr(w, problems.InternalError())
		return
	}
	if deposit == nil {
		ape.RenderErr(w, problems.NotFound())
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

	response := common.ToStatusResponse(deposit)
	raw := common.ProtoJsonMustMarshal(response)
	if err = ws.WriteMessage(websocket.TextMessage, raw); err != nil {
		logger.WithError(err).Error("failed to write message to websocket")
		_ = ws.Close()
		return
	}

	// no websocket needed for non-changeable statuses
	if slices.Contains(database.FinalWithdrawalStatuses, deposit.WithdrawalStatus) {
		_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = ws.Close()
		return
	}

	watchWithdrawalStatus(ctxt, ws, gracefulClose, *deposit)
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

func watchWithdrawalStatus(ctxt context.Context, ws *websocket.Conn, connClosed chan struct{}, deposit database.Deposit) {
	defer func() { _ = ws.Close() }()

	var (
		db         = ctx.DB(ctxt)
		logger     = ctx.Logger(ctxt)
		prevStatus = deposit.WithdrawalStatus
		ticker     = time.NewTicker(pollingPeriod)
	)

	defer ticker.Stop()

	for {
		select {
		case <-connClosed:
			return
		case <-ctxt.Done():
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"))
			return
		case <-ticker.C:
			// doing nothing, just waiting some period
		}

		withdrawal, err := db.Get(deposit.DepositIdentifier)
		if err != nil {
			logger.WithError(err).Error("failed to get withdrawal")
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Internal server error"))
			return
		}

		//poll until our new status is different from the previous one
		if withdrawal.WithdrawalStatus == prevStatus {
			continue
		}

		response := common.ToStatusResponse(withdrawal)
		raw := common.ProtoJsonMustMarshal(response)
		if err = ws.WriteMessage(websocket.TextMessage, raw); err != nil {
			logger.WithError(err).Error("failed to write message to websocket")
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Internal server error"))
			return
		}

		// is it a time for websocket closing
		if slices.Contains(database.FinalWithdrawalStatuses, withdrawal.WithdrawalStatus) {
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}

		prevStatus = withdrawal.WithdrawalStatus
	}
}

func identifierFromParams(r *http.Request) (*database.DepositIdentifier, error) {
	identifier := &database.DepositIdentifier{
		ChainId: chi.URLParam(r, paramChainId),
		TxHash:  chi.URLParam(r, paramTxHash),
	}

	nonce, err := strconv.ParseInt(chi.URLParam(r, paramTxNonce), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tx nonce")
	}
	if nonce < 0 {
		return nil, errors.New("tx nonce cannot be negative")
	}

	identifier.TxNonce = nonce

	return identifier, nil
}

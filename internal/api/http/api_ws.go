package http

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/api/common"
	"github.com/Bridgeless-Project/tss-svc/internal/api/ctx"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/gorilla/websocket"
	"gitlab.com/distributed_lab/ape"
	"gitlab.com/distributed_lab/ape/problems"
)

const pollingPeriod = 1 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func CheckWithdrawalWs(w http.ResponseWriter, r *http.Request) {
	var (
		ctxt        = r.Context()
		logger      = ctx.Logger(ctxt)
		db          = ctx.DB(ctxt)
		clientsRepo = ctx.Clients(ctxt)
	)

	identifier, err := common.IdentifierFromParams(r)
	if err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		return
	}
	client, err := clientsRepo.Client(identifier.ChainId)
	if err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		return
	}
	if err = common.ValidateChainIdentifier(common.FromDbIdentifier(*identifier), client); err != nil {
		ape.RenderErr(w, problems.BadRequest(err)...)
		return
	}

	deposit, err := db.Get(*identifier)
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

	rawMsg := common.ProtoJsonMustMarshal(common.ToStatusResponse(&deposit))
	if err := ws.WriteMessage(websocket.TextMessage, rawMsg); err != nil {
		_ = ws.Close()
		return
	}

	if slices.Contains(database.FinalWithdrawalStatuses, deposit.WithdrawalStatus) {
		_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = ws.Close()
		return
	}

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

		// poll until our new status is different from the previous one
		if withdrawal.WithdrawalStatus == prevStatus {
			continue
		}

		rawMsg = common.ProtoJsonMustMarshal(common.ToStatusResponse(&deposit))
		if err = ws.WriteMessage(websocket.TextMessage, rawMsg); err != nil {
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

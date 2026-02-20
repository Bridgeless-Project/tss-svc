package utxo

import (
	"context"
	"time"

	utxoclient "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/client"
	utxoutils "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ resharingTypes.Handler = &Handler{}

type Handler struct {
	client         utxoclient.Client
	sessionManager *p2p.SessionManager
	parties        []p2p.Party
	self           tss.LocalSignParty
	logger         *logan.Entry

	unspentCount  uint
	sessionsCount uint
}

func NewHandler(
	self tss.LocalSignParty,
	parties []p2p.Party,
	client utxoclient.Client,
	sessionManager *p2p.SessionManager,
	logger *logan.Entry,
) *Handler {
	handler := &Handler{
		client:         client,
		sessionManager: sessionManager,
		parties:        parties,
		self:           self,
		logger:         logger,
	}

	if err := handler.init(); err != nil {
		panic(errors.Wrap(err, "failed to initialize utxo resharing handler"))
	}

	return handler
}

func (h *Handler) init() error {
	unspent, err := h.client.ListUnspent()
	if err != nil {
		return errors.Wrap(err, "failed to list unspent outputs")
	}

	h.unspentCount = uint(len(unspent))

	maxUnspentPerSession := utxoutils.DefaultResharingParams.SetParams[0].MaxInputsCount
	sessionsCount := (h.unspentCount + maxUnspentPerSession - 1) / maxUnspentPerSession // rounding up

	h.sessionsCount = sessionsCount

	return nil
}

func (h *Handler) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	// TODO: check count of UTXOs controlled by old key

	return false, nil
}

func (h *Handler) MaxHandleDuration() time.Duration {
	maxUnspentPerSession := utxoutils.DefaultResharingParams.SetParams[0].MaxInputsCount

	// assuming maximum possible duration without checking the unspent count in the last session
	return time.Duration(h.sessionsCount) * MaxSessionDuration(maxUnspentPerSession)
}

func (h *Handler) Handle(ctx context.Context, state *resharingTypes.State) error {
	var (
		maxInputsCount  = utxoutils.DefaultResharingParams.SetParams[0].MaxInputsCount
		inputsLeft      = h.unspentCount
		resharingParams = utxoutils.DefaultResharingParams
		targetAddr      = h.client.UtxoHelper().P2pkhAddress(state.NewPubKey)
	)

	// FIXME: remove
	resharingParams.SetParams[0].MaxInputsCount = 5

	nextStartTime := state.SessionStartTime
	for idx := range h.sessionsCount {
		// last session might have fewer inputs than maxUnspentPerSession,
		// so we need to recalculate outs count
		if idx == h.sessionsCount-1 {
			// proportionally calculate outs count based on inputs left and max inputs count
			maxOutsCount := utxoutils.DefaultResharingParams.SetParams[0].OutsCount
			outsCount := (inputsLeft*maxOutsCount + maxInputsCount - 1) / maxInputsCount // rounding up
			resharingParams.SetParams[0].OutsCount = outsCount
		}

		sessParams := SessionParams{
			SessionParams: session.Params{
				Id:        int64(idx + 1),
				Threshold: h.self.Threshold,
			},
			TargetAddr:        targetAddr,
			ConsolidateParams: resharingParams,
		}

		session := NewSession(
			h.self,
			h.client,
			sessParams,
			h.parties,
			h.logger.WithField("component", "utxo_resharing_handler").WithField("session_id", idx+1),
		)
		h.sessionManager.Add(session)
		<-time.After(time.Second) // slight delay to ensure session is registered before first message arrives

		nextStartTime = nextStartTime.Add(MaxSessionDuration(maxInputsCount) + time.Second)
		if err := session.Run(ctx); err != nil {
			return errors.Wrapf(err, "failed to run resharing session %d", idx+1)
		}
		_, err := session.WaitFor()
		if err != nil {
			return errors.Wrapf(err, "resharing session %d failed", idx+1)
		}

		if idx == h.sessionsCount-1 {
			break
		} else {
			inputsLeft -= maxInputsCount
		}

		// some parties won't participate in resharing, so we wait until start time to ensure synchronization
		<-time.After(time.Until(nextStartTime))
	}

	state.AddBridgeAddress(h.client.ChainId(), targetAddr)

	return nil
}

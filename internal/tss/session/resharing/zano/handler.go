package zano

import (
	"context"
	"slices"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/zano"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	resharingTypes "github.com/Bridgeless-Project/tss-svc/internal/tss/session/resharing/types"
	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ resharingTypes.Handler = &Handler{}

type Handler struct {
	connector      *connector.Connector
	client         *zano.Client
	sessionManager *p2p.SessionManager
	parties        []p2p.Party
	logger         *logan.Entry
	self           tss.LocalSignParty

	assetInfos      map[string]zanoTypes.AssetDescriptor
	resharingAssets []string
}

func NewHandler(
	self tss.LocalSignParty,
	parties []p2p.Party,
	client *zano.Client,
	sessionManager *p2p.SessionManager,
	logger *logan.Entry,
	connector *connector.Connector,
) *Handler {
	handler := &Handler{
		connector:      connector,
		client:         client,
		parties:        parties,
		self:           self,
		logger:         logger,
		sessionManager: sessionManager,
		assetInfos:     make(map[string]zanoTypes.AssetDescriptor),
	}

	if err := handler.fetchAssetInfos(); err != nil {
		panic(errors.Wrap(err, "failed to configure resharing tokens"))
	}

	return handler
}

func (h *Handler) getResharingAssets(pubkeyHex string) []string {
	if h.resharingAssets != nil {
		return h.resharingAssets
	}

	var resharingAssets []string
	for assetId, assetInfo := range h.assetInfos {
		if assetInfo.OwnerEthPubKey != pubkeyHex {
			resharingAssets = append(resharingAssets, assetId)
		}
	}

	slices.Sort(resharingAssets)
	h.resharingAssets = resharingAssets

	return resharingAssets
}

func (h *Handler) MaxHandleDuration() time.Duration {
	return time.Duration(len(h.assetInfos)) * (SessionMaxDuration + time.Second)
}

func (h *Handler) RecoverStateIfProcessed(state *resharingTypes.State) (bool, error) {
	newPubKeyHex := hexutil.Encode(crypto.CompressPubkey(state.NewPubKey))
	if len(h.getResharingAssets(newPubKeyHex)) == 0 {
		return true, nil
	}

	return false, nil
}

func (h *Handler) Handle(ctx context.Context, state *resharingTypes.State) error {
	var (
		newPubKeyHex    = hexutil.Encode(crypto.CompressPubkey(state.NewPubKey))
		assetsToReshare = h.getResharingAssets(newPubKeyHex)
	)

	nextStartTime := state.SessionStartTime
	for idx, assetId := range assetsToReshare {
		sessParams := SessionParams{
			SessionParams: session.Params{
				Id:        int64(idx + 1),
				Threshold: h.self.Threshold,
			},
			AssetId:     assetId,
			OwnerPubKey: newPubKeyHex,
			IsEthKey:    true,
		}
		session := NewSession(
			h.self,
			h.client,
			sessParams,
			h.parties,
			h.logger.WithField("component", "zano_resharing_handler").WithField("asset", assetId),
		)
		h.sessionManager.Add(session)
		<-time.After(time.Second) // slight delay to ensure session is registered before first message arrives

		nextStartTime = nextStartTime.Add(session.MaxDuration() + time.Second)
		if err := session.Run(ctx); err != nil {
			return errors.Wrapf(err, "failed to run resharing session for asset %s", assetId)
		}
		_, err := session.WaitFor()
		if err != nil {
			return errors.Wrapf(err, "resharing session for asset %s failed", assetId)
		}

		if idx == len(assetsToReshare)-1 {
			break
		}

		// some parties won't participate in resharing, so we wait until start time to ensure synchronization
		<-time.After(time.Until(nextStartTime))
	}

	return nil
}

func (h *Handler) fetchAssetInfos() error {
	tokens, err := h.connector.GetTokens()
	if err != nil {
		return errors.Wrap(err, "failed to get all tokens")
	}

	var allZanoAssets []string
	for _, token := range tokens {
		for _, tokenInfo := range token.Info {
			if tokenInfo.ChainId == h.client.ChainId() {
				allZanoAssets = append(allZanoAssets, tokenInfo.Address)
			}
		}
	}
	if len(allZanoAssets) == 0 {
		return nil
	}

	for _, asset := range allZanoAssets {
		info, err := h.client.GetAssetInfo(asset)
		if err != nil {
			return errors.Wrapf(err, "failed to get asset info for %s", asset)
		}
		h.assetInfos[asset] = *info
	}

	return nil
}

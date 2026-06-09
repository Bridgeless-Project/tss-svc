package commissions

import (
	"cmp"
	"context"
	"math/big"
	"slices"
	"strconv"
	"time"

	bridgeKeeper "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/keeper"
	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
	bridgeconfig "github.com/Bridgeless-Project/tss-svc/internal/bridge/config"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/signing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

type Session struct {
	event     core.EventCommissionCollection
	connector *connector.Connector
	settings  bridgeconfig.EvmSettings

	sessionManager *p2p.SessionManager
	self           tss.LocalSignParty
	parties        []p2p.Party

	logger *logan.Entry
}

func NewSession(
	event core.EventCommissionCollection,
	self tss.LocalSignParty,
	parties []p2p.Party,
	connector *connector.Connector,
	settings bridgeconfig.EvmSettings,
	sessionManager *p2p.SessionManager,
	logger *logan.Entry,
) *Session {
	return &Session{
		event:     event,
		connector: connector,
		settings:  settings,

		sessionManager: sessionManager,
		self:           self,
		parties:        parties,

		logger: logger,
	}
}

func (s *Session) Run(ctx context.Context) ([]bridgeTypes.SystemWithdrawal, error) {
	commissionsData, err := s.loadCommissionData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load commission data")
	} else if len(commissionsData) == 0 {
		// no commissions to collect, returning empty result
		return nil, nil
	}

	var (
		withdrawals   = make([]bridgeTypes.SystemWithdrawal, 0, len(commissionsData))
		sessStartTime = s.event.Time.Add(session.CommissionCollectionSessionDelay)
	)
	for _, data := range commissionsData {
		signAttempts := 1

		for {
			if signAttempts > 5 {
				return nil, errors.Errorf("max attempts reached for token %v to sign commission withdrawal, skipping it", data.DestinationToken)
			}

			sessionId := data.TxHash + strconv.Itoa(signAttempts) // unique for each signing attempt
			sess := signing.NewConsensusSession(
				sessionId,
				s.self,
				s.parties,
				consensus.NewPlainSignDataMechanism(data.ToOperation().CalculateHashPrefixed()),
				s.logger.WithField("component", "commission_session"),
			)
			s.sessionManager.Add(sess)

			s.logger.Infof("signing commission withdrawal for token %v will start in %v", data.DestinationToken, time.Until(sessStartTime))
			select {
			case <-ctx.Done():
				s.sessionManager.Remove(sess.Id())
				return nil, errors.Wrap(ctx.Err(), "session context done while waiting to start commission collection session")
			case <-time.After(time.Until(sessStartTime)):
				signAttempts++
				sessStartTime = sessStartTime.Add(session.BoundaryConsensusSession)
			}

			err = sess.Run(ctx)
			s.sessionManager.Remove(sess.Id())
			if err != nil {
				s.logger.WithError(err).Errorf("failed to sign commission data for token %v", data.DestinationToken)
				continue
			}

			s.logger.Infof("commission withdrawal for token %v signed successfully", data.DestinationToken)

			result := sess.Result()
			signature := evm.ConvertSignature(result.Signature)

			withdrawals = append(withdrawals, bridgeTypes.SystemWithdrawal{
				TxHash:    data.TxHash,
				TxIndex:   uint64(data.TxNonce),
				Block:     uint64(s.event.BlockHeight),
				Receiver:  data.Receiver.String(),
				Token:     data.DestinationToken.String(),
				Signature: signature,
				IsWrapped: data.IsWrapped,
				Amount:    data.WithdrawalAmount.String(),
				EpochId:   s.event.EpochId,
			})

			break
		}

		signAttempts = 1
	}

	return withdrawals, nil
}

func (s *Session) loadCommissionData() ([]operations.WithdrawOperationData, error) {
	tokens, err := s.connector.GetTokens(s.event.BlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all tokens")
	}

	slices.SortFunc(tokens, func(a, b bridgeTypes.Token) int { return cmp.Compare(a.Id, b.Id) })

	ops := make([]operations.WithdrawOperationData, 0)
	for _, token := range tokens {
		bridgeChainIdx := slices.IndexFunc(token.Info, func(t bridgeTypes.TokenInfo) bool { return t.ChainId == s.settings.ChainId })
		if bridgeChainIdx == -1 {
			// token is not bridgeable to the chain we operate on, skipping it
			continue
		}
		bridgeTokenInfo := token.Info[bridgeChainIdx]

		tokenCommission, err := s.connector.GetCommission(s.event.EpochId, token.Id, s.event.BlockHeight)
		if err != nil {
			if errors.Is(err, core.ErrCommissionNotFound) {
				// no commission to collect for this token, skipping it
				continue
			}

			return nil, errors.Wrapf(err, "failed to get commission for token %v", token.Id)
		}

		amount, set := new(big.Int).SetString(tokenCommission.Amount, 10)
		if !set {
			return nil, errors.Errorf("failed to parse commission amount %s for token %v", tokenCommission.Amount, token.Id)
		} else if amount.Cmp(bridge.ZeroAmount) <= 0 {
			// no commission to collect for this token, skipping it
			continue
		}

		ops = append(ops, operations.WithdrawOperationData{
			WithdrawalAmount: amount,
			Receiver:         bridgeTypes.ModuleAddress,
			TxHash: bridgeKeeper.ConstructSystemTxHash(
				amount,
				common.HexToAddress(bridgeTokenInfo.Address).Bytes(),
				bridgeTypes.ModuleAddress.Bytes(),
			),
			TxNonce:          0,
			ChainId:          s.settings.ChainIdAsBigInt(),
			DestinationToken: common.HexToAddress(bridgeTokenInfo.Address),
			IsWrapped:        bridgeTokenInfo.IsWrapped,
		})
	}

	return ops, nil
}

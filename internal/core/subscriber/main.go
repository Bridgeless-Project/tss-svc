package subscriber

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	database "github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	bridgeTypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/distributed_lab/logan/v3"
)

const (
	OpServiceName = "op-subscriber"
	OpPoolSize    = 50
)

type SubmitEventSubscriber struct {
	db        database.DepositsQ
	client    *http.HTTP
	query     string
	log       *logan.Entry
	connector *connector.Connector
}

func NewSubmitEventSubscriber(db database.DepositsQ, client *http.HTTP, logger *logan.Entry, connector *connector.Connector) *SubmitEventSubscriber {
	return &SubmitEventSubscriber{
		db:     db,
		client: client,
		log:    logger,
		query: fmt.Sprintf("tm.event='Tx' AND %s.%s EXISTS",
			bridgeTypes.EventType_DEPOSIT_SUBMITTED.String(),
			bridgeTypes.AttributeKeyDepositTxHash,
		),
		connector: connector,
	}
}

func (s *SubmitEventSubscriber) Run(ctx context.Context) error {
	out, err := s.client.Subscribe(ctx, OpServiceName, s.query, OpPoolSize)
	if err != nil {
		return errors.Wrap(err, "subscriber init failed")
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		s.run(ctx, out)
		wg.Done()
	}()
	go func() {
		s.runSubmitter(ctx)
		wg.Done()
	}()

	wg.Wait()

	return nil
}

func (s *SubmitEventSubscriber) runSubmitter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.log.Info("context cancelled, stopping transaction submitter")
			return
		default:
			pendingDeposit, err := s.db.GetWithSelector(database.DepositsSelector{
				NotSubmitted: true,
				One:          true,
			})
			if err != nil {
				s.log.WithError(err).Error("failed to get pending submit")
				time.Sleep(5 * time.Second)
				continue
			} else if pendingDeposit == nil {
				time.Sleep(5 * time.Second)
				continue
			}

			s.log.WithField("deposit", pendingDeposit.DepositIdentifier.TxHash).Info("got deposit to submit")

			submitted := false
			for i := 0; i < 5; i++ {
				err = s.connector.SubmitDeposits(ctx, pendingDeposit.ToTransaction())
				if err == nil || errors.Is(err, core.ErrTransactionAlreadySubmitted) {
					submitted = true
					s.log.WithField("deposit", pendingDeposit.DepositIdentifier.TxHash).Info("deposit submitted")
					if err = s.db.UpdateSubmittedStatus(pendingDeposit.DepositIdentifier, true); err != nil {
						s.log.WithError(err).Error("failed to update deposit as submitted")
					}
					break
				}

				s.log.WithError(err).WithField("attempt", i+1).Error("failed to submit deposit, retrying")
				time.Sleep(1 * time.Second)
			}

			if !submitted {
				s.log.WithField("deposit", pendingDeposit.DepositIdentifier.TxHash).Error("failed to submit deposit after retries, will retry later")
				time.Sleep(10 * time.Second)
			}
		}
	}
}

func (s *SubmitEventSubscriber) run(ctx context.Context, out <-chan coretypes.ResultEvent) {
	for {
		select {
		case <-ctx.Done():
			s.log.Info("context cancelled, stopping receiving events")
			shutdownDeadline, cancel := context.WithTimeout(context.Background(), time.Second)
			if err := s.client.Unsubscribe(shutdownDeadline, OpServiceName, s.query); err != nil {
				s.log.WithError(err).Error("failed to unsubscribe from new operations")
			}

			cancel()
			return
		case c, ok := <-out:
			if !ok {
				s.log.Warn("chanel closed, stopping receiving messages")
				return
			}

			s.log.Info("received new event")
			eventDeposit, err := parseSubmittedDeposit(c.Events)
			if err != nil {
				s.log.WithError(err).Error("failed to parse submitted deposit")
				continue
			}

			existingDeposit, err := s.db.Get(eventDeposit.DepositIdentifier)
			if err != nil {
				s.log.WithError(err).Error("failed to get deposit")
				continue
			}

			if existingDeposit == nil {
				s.log.Info("found new submitted deposit")
				if _, err = s.db.InsertProcessedDeposit(*eventDeposit); err != nil {
					s.log.WithError(err).Error("failed to insert new deposit")
				}
				continue
			}

			switch existingDeposit.WithdrawalStatus {
			case types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSED:
				s.log.Info("skipping processed deposit")
			case types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSING,
				types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING:
				s.log.Info("found new deposit data to update")
				if err = s.db.UpdateWithdrawalDetails(
					existingDeposit.DepositIdentifier,
					eventDeposit.WithdrawalTxHash,
					eventDeposit.Signature,
				); err != nil {
					s.log.WithError(err).Error("failed to update deposit withdrawal details")
				}
				s.log.Info("deposit withdrawal details successfully updated")
			default:
				s.log.Infof("nothing to do with deposit status %s", existingDeposit.WithdrawalStatus)
			}
		}
	}
}

func parseSubmittedDeposit(attributes map[string][]string) (*database.Deposit, error) {
	deposit := &database.Deposit{}

	for keys, attribute := range attributes {
		parts := strings.SplitN(keys, ".", 2)
		if parts[0] != bridgeTypes.EventType_DEPOSIT_SUBMITTED.String() {
			// skip if not deposit submitted event
			continue
		}

		switch parts[1] {
		case bridgeTypes.AttributeKeyDepositTxHash:
			deposit.TxHash = attribute[0]
		case bridgeTypes.AttributeKeyDepositNonce:
			n, err := strconv.ParseInt(attribute[0], 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse deposit nonce")
			}
			deposit.TxNonce = n
		case bridgeTypes.AttributeKeyDepositChainId:
			deposit.ChainId = attribute[0]
		case bridgeTypes.AttributeKeyDepositAmount:
			deposit.DepositAmount = attribute[0]
		case bridgeTypes.AttributeKeyDepositToken:
			deposit.DepositToken = attribute[0]
		case bridgeTypes.AttributeKeyDepositBlock:
			b, err := strconv.ParseInt(attribute[0], 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse deposit block")
			}
			deposit.DepositBlock = b
		case bridgeTypes.AttributeKeyWithdrawalAmount:
			deposit.WithdrawalAmount = attribute[0]
		case bridgeTypes.AttributeKeyDepositor:
			deposit.Depositor = &attribute[0]
		case bridgeTypes.AttributeKeyReceiver:
			deposit.Receiver = attribute[0]
		case bridgeTypes.AttributeKeyWithdrawalChainID:
			deposit.WithdrawalChainId = attribute[0]
		case bridgeTypes.AttributeKeyWithdrawalTxHash:
			if attribute[0] != "" {
				deposit.WithdrawalTxHash = &attribute[0]
			}
		case bridgeTypes.AttributeKeyWithdrawalToken:
			deposit.WithdrawalToken = attribute[0]
		case bridgeTypes.AttributeKeySignature:
			if attribute[0] != "" {
				deposit.Signature = &attribute[0]
			}
		case bridgeTypes.AttributeKeyIsWrapped:
			isWrapped, err := strconv.ParseBool(attribute[0])
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse isWrapped attribute")
			}
			deposit.IsWrappedToken = isWrapped
		case bridgeTypes.AttributeKeyCommissionAmount:
			deposit.CommissionAmount = attribute[0]
		default:
			return nil, errors.Wrap(errors.New(fmt.Sprintf("unknown attribute key: %s", parts[1])), "failed to parse attribute")
		}
	}

	return deposit, nil
}

package subscriber

import (
	"context"
	"fmt"
	bridgeTypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	database "github.com/hyle-team/tss-svc/internal/db"
	"github.com/hyle-team/tss-svc/internal/types"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	"strconv"
	"strings"
)

const (
	OpServiceName = "op-subscriber"
	OpPoolSize    = 1000

	OpQuerySubmit = "tm.event='Tx' AND message.action='/core.bridge.MsgSubmitTransactions'"
)

type Subscriber struct {
	db     database.DepositsQ
	client *http.HTTP
	query  string
	log    *logan.Entry
}

func NewSubmitSubscriber(db database.DepositsQ, client *http.HTTP, logger *logan.Entry) *Subscriber {
	return &Subscriber{
		db:     db,
		client: client,
		query:  OpQuerySubmit,
		log:    logger,
	}
}

func (s *Subscriber) Run(ctx context.Context) {
	s.log.Info("starting subscriber")
	s.log.Info("Query is ", s.query)
	out, err := s.client.Subscribe(ctx, OpServiceName, s.query, OpPoolSize)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				if err := s.client.Unsubscribe(ctx, OpServiceName, s.query); err != nil {
					s.log.WithError(err).Error("Failed to unsubscribe from new operations")
				}

				s.log.Info("Context finished")
				return
			case c, ok := <-out:
				if !ok {
					s.log.Fatal("chanel closed")
				}
				deposit, err := parseSubmittedDeposit(c.Events)
				if err != nil {
					s.log.WithError(err).Error("Failed to parse submitted deposit")
					continue
				}
				tx, err := s.db.Get(deposit.DepositIdentifier)
				if err != nil {
					s.log.WithError(err).Error("Failed to get deposit")
					continue
				}

				// if deposit does not exist in db insert it
				if tx == nil {
					s.log.Info("Found new submitted deposit")
					if _, err = s.db.Insert(*deposit); err != nil {
						s.log.WithError(err).Error("Failed to insert deposit")
						continue
					}
					if err = s.db.UpdateWithdrawalDetails(deposit.DepositIdentifier, *deposit.WithdrawalTxHash, *deposit.Signature); err != nil {
						s.log.WithError(err).Error("Failed to update deposit withdrawal details")
						continue
					}
					continue
				}

				// if deposit exists and pending or processing update signature,withdrawal tx hash and status
				s.log.Info("Found existing deposit submitted to core")
				if tx.WithdrawalStatus == types.WithdrawalStatus_WITHDRAWAL_STATUS_PROCESSING || tx.WithdrawalStatus == types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING {
					if err = s.db.UpdateWithdrawalDetails(tx.DepositIdentifier, *deposit.WithdrawalTxHash, *deposit.Signature); err != nil {
						s.log.WithError(err).Error("Failed to update deposit withdrawal details")
						continue
					}
				}

			}
		}
	}()
}

func parseSubmittedDeposit(attributes map[string][]string) (*database.Deposit, error) {
	deposit := &database.Deposit{}
	for keys, attribute := range attributes {

		parts := strings.SplitN(keys, ".", 2)
		if parts[0] != bridgeTypes.EventType_DEPOSIT_SUBMITTED.String() {
			continue
		}

		switch parts[1] {
		case bridgeTypes.AttributeKeyDepositTxHash:
			deposit.TxHash = attribute[0]
		case bridgeTypes.AttributeKeyDepositNonce:
			n, err := strconv.Atoi(attribute[0])
			if err != nil {
				return nil, errors.Wrap(errors.New(fmt.Sprintf("got invalid nonce, got %s", attribute)), "invalid nonce")
			}
			deposit.TxNonce = n
		case bridgeTypes.AttributeKeyDepositChainId:
			deposit.ChainId = attribute[0]
		case bridgeTypes.AttributeKeyDepositAmount:
			deposit.DepositAmount = &attribute[0]
		case bridgeTypes.AttributeKeyDepositToken:
			deposit.DepositToken = &attribute[0]
		case bridgeTypes.AttributeKeyDepositBlock:
			b, err := strconv.ParseInt(attribute[0], 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse deposit block")
			}
			deposit.DepositBlock = &b
		case bridgeTypes.AttributeKeyWithdrawalAmount:
			deposit.WithdrawalAmount = &attribute[0]
		case bridgeTypes.AttributeKeyDepositor:
			deposit.Depositor = &attribute[0]
		case bridgeTypes.AttributeKeyReceiver:
			deposit.Receiver = &attribute[0]
		case bridgeTypes.AttributeKeyWithdrawalChainID:
			deposit.WithdrawalChainId = &attribute[0]
		case bridgeTypes.AttributeKeyWithdrawalTxHash:
			deposit.WithdrawalTxHash = &attribute[0]
		case bridgeTypes.AttributeKeyWithdrawalToken:
			deposit.WithdrawalToken = &attribute[0]
		case bridgeTypes.AttributeKeySignature:
			deposit.Signature = &attribute[0]
		case bridgeTypes.AttributeKeyIsWrapped:
			isWrapped, err := strconv.ParseBool(attribute[0])
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse isWrapped attribute")
			}
			deposit.IsWrappedToken = &isWrapped
		default:

			return nil, errors.Wrap(errors.New(fmt.Sprintf("unknown attribute key: %s", parts[1])), "failed to parse attribute")
		}
	}

	return deposit, nil
}

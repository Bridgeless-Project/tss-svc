package centralized

import (
	"context"
	"time"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/Bridgeless-Project/tss-svc/internal/p2p"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session"
	"github.com/Bridgeless-Project/tss-svc/internal/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/logan/v3"
)

var _ p2p.TssSession = &Session{}

type Session struct {
	id string

	client *evm.Client
	db     db.DepositsQ

	logger *logan.Entry
}

func NewSession(
	client *evm.Client,
	db db.DepositsQ,
	logger *logan.Entry,
) *Session {
	id := session.GetDefaultSigningSessionIdentifier(client.ChainId())
	return &Session{
		id:     id,
		client: client,
		db:     db,
		logger: logger.WithField("session_id", id),
	}
}

func (s *Session) Id() string {
	return s.id
}

// Receive is a no-op for centralized signing sessions.
func (s *Session) Receive(request *p2p.SubmitRequest) error {
	return nil
}

// SigningSessionInfo mocks signing session info for centralized signing sessions.
func (s *Session) SigningSessionInfo() *p2p.SigningSessionInfo {
	return &p2p.SigningSessionInfo{ChainId: s.client.ChainId(), NextSessionStartTime: 1}
}

// RegisterIdChangeListener is a no-op for centralized signing sessions.
func (s *Session) RegisterIdChangeListener(func(oldId, newId string)) {}

func (s *Session) Run(ctx context.Context) error {
	s.logger.Info("signing session started")

	var (
		statusPending = types.WithdrawalStatus_WITHDRAWAL_STATUS_PENDING
		chainId       = s.client.ChainId()
		cooldown      = time.Second * 0
		batchSize     = uint64(100)
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("signing session cancelled")
			return nil
		case <-time.After(cooldown):
			cooldown = time.Second * 1
			deposits, err := s.db.Select(db.DepositsSelector{
				ChainId:       &chainId,
				Status:        &statusPending,
				Distributed:   true,
				SortAscending: true,
				Limit:         batchSize,
			})
			switch {
			case err != nil:
				s.logger.WithError(err).Error("failed to fetch pending deposits")
				fallthrough
			case len(deposits) == 0:
				continue
			default:
				s.logger.Infof("fetched %d pending deposits", len(deposits))
			}

			start := time.Now()
			if err = s.processDeposits(deposits); err != nil {
				s.logger.WithError(err).Error("failed to process deposits")
				continue
			}
			s.logger.Infof("signed and updated %d deposits in %s", len(deposits), time.Since(start))

			cooldown = time.Second * 0
		}
	}
}

func (s *Session) processDeposits(deposits []db.Deposit) error {
	var signedDeposits = make([]db.SignedDeposit, len(deposits))

	for i, deposit := range deposits {
		signature, err := s.client.Sign(deposit)
		if err != nil {
			return errors.Wrapf(err, "failed to sign deposit id %d", deposit.Id)
		}

		signedDeposits[i] = db.SignedDeposit{Id: deposit.Id, Signature: hexutil.Encode(signature)}
	}
	if err := s.db.UpdateSignedBatch(signedDeposits); err != nil {
		return errors.Wrap(err, "failed to update signed deposits")
	}

	return nil
}

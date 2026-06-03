package commissions

import (
	"fmt"
	"math/big"

	bridgeTypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/operations"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	"github.com/Bridgeless-Project/tss-svc/internal/core/connector"
	"github.com/Bridgeless-Project/tss-svc/internal/tss/session/consensus"
	"github.com/pkg/errors"
)

var (
	_ consensus.SigningData            = &SigningData{}
	_ consensus.Mechanism[SigningData] = &Constructor{}
)

type SigningData struct {
	signHash []byte
}

func (s SigningData) HashString() string {
	return fmt.Sprintf("%v", s.signHash)
}

type Constructor struct {
	data      core.EventDataCommissionCollection
	tokenId   uint64
	tokenInfo bridgeTypes.TokenInfo
	connector *connector.Connector
}

func (c Constructor) FormProposalData() (*SigningData, error) {
	commission, err := c.connector.GetCommission(c.data.EpochId, c.tokenId)
	if err != nil {
		if errors.Is(err, core.ErrCommissionNotFound) {
			// no data to withdraw for this token and epoch
			return nil, nil
		}

		return nil, errors.Wrapf(err, "failed to get commission for token id %d and epoch id %d", c.tokenId, c.data.EpochId)
	}

	amount, parsed := new(big.Int).SetString(commission.Amount, 10)
	if !parsed {
		return nil, errors.Errorf("invalid commission amount %s for token id %d and epoch id %d", commission.Amount, c.tokenId, c.data.EpochId)
	} else if amount.Cmp(bridge.ZeroAmount) == 0 {
		// no data to withdraw for this token and epoch
		return nil, nil
	}

	var operation evm.Operation
	if c.tokenInfo.Address == bridge.DefaultNativeTokenAddress {
		// todo
	} else {
		operation = operations.WithdrawERC20OperationData{
			WithdrawalAmount: amount,
			Receiver:         bridgeTypes.ModuleAddress,
			TxHash:           "TODO",
			TxNonce:          0,
		}.ToOperation()
	}

	return &SigningData{}, nil
}

func (c Constructor) VerifyProposedData(data SigningData) error {
	return nil
}

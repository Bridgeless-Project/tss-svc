package evm

import (
	"context"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/pkg/errors"
)

func (p *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	txReceipt, from, err := p.GetTransactionReceipt(common.HexToHash(id.TxHash))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction receipt")
	}

	if txReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, bridgeTypes.ErrTxFailed
	}

	if int64(len(txReceipt.Logs)) < id.TxNonce+1 {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	log := txReceipt.Logs[id.TxNonce]
	if log.Address.Hex() != p.chain.BridgeAddress.Hex() {
		return nil, bridgeTypes.ErrUnsupportedContract
	}

	depositType := p.getDepositLogType(log)
	if depositType == "" {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	if err = p.validateConfirmations(txReceipt); err != nil {
		return nil, errors.Wrap(err, "failed to validate confirmations")
	}

	var unpackedData *db.DepositData

	switch depositType {
	case EventDepositedNative:
		eventBody := new(contracts.BridgeDepositedNative)
		if err = p.contractABI.UnpackIntoInterface(eventBody, depositType, log.Data); err != nil {
			return nil, bridgeTypes.ErrDepositNotFound
		}

		unpackedData = &db.DepositData{
			DepositIdentifier:  id,
			DestinationChainId: eventBody.Network,
			DestinationAddress: eventBody.Receiver,
			TokenAddress:       bridge.DefaultNativeTokenAddress,
			DepositAmount:      eventBody.Amount,
			Block:              int64(log.BlockNumber),
			SourceAddress:      from.String(),
		}

		break

	case EventDepositedERC20:
		eventBody := new(contracts.BridgeDepositedERC20)
		if err = p.contractABI.UnpackIntoInterface(eventBody, depositType, log.Data); err != nil {
			return nil, bridgeTypes.ErrDepositNotFound
		}

		unpackedData = &db.DepositData{
			DepositIdentifier:  id,
			DestinationChainId: eventBody.Network,
			DestinationAddress: eventBody.Receiver,
			DepositAmount:      eventBody.Amount,
			TokenAddress:       strings.ToLower(eventBody.Token.String()),
			Block:              int64(log.BlockNumber),
			SourceAddress:      from.String(),
		}

		break
	default:
		return nil, bridgeTypes.ErrUnsupportedEvent
	}

	if unpackedData == nil {
		return nil, bridgeTypes.ErrFailedUnpackLogs
	}

	return unpackedData, nil
}

func (p *Client) validateConfirmations(receipt *types.Receipt) error {
	curHeight, err := p.chain.Rpc.BlockNumber(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to get current block number")
	}

	// including the current block
	if receipt.BlockNumber.Uint64()+p.chain.Confirmations-1 > curHeight {
		return bridgeTypes.ErrTxNotConfirmed
	}

	return nil
}

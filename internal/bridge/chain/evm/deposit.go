package evm

import (
	"context"
	"strings"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	v1 "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v1"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/evm/contracts/v2"
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

	eventType := p.GetDepositEventType(log)
	if eventType == "" {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	if err = p.validateConfirmations(txReceipt); err != nil {
		return nil, errors.Wrap(err, "failed to validate confirmations")
	}

	var unpackedData *db.DepositData
	switch eventType {
	case EventV1DepositedNative:
		eventBody := new(v1.BridgeDepositedNative)
		if err = p.abiV1.UnpackIntoInterface(eventBody, EventToEventName[eventType], log.Data); err != nil {
			return nil, bridgeTypes.ErrFailedUnpackLogs
		}
		unpackedData = &db.DepositData{
			DepositIdentifier:  id,
			DestinationChainId: eventBody.Network,
			DestinationAddress: eventBody.Receiver,
			TokenAddress:       bridge.DefaultNativeTokenAddress,
			DepositAmount:      eventBody.Amount,
			Block:              int64(log.BlockNumber),
			SourceAddress:      from.String(),
			ReferralId:         0, // v1 does not have referralId
		}
	case EventV2DepositedNative:
		eventBody := new(v2.BridgeDepositedNative)
		if err = p.abiV2.UnpackIntoInterface(eventBody, EventToEventName[eventType], log.Data); err != nil {
			return nil, bridgeTypes.ErrFailedUnpackLogs
		}
		unpackedData = &db.DepositData{
			DepositIdentifier:  id,
			DestinationChainId: eventBody.Network,
			DestinationAddress: eventBody.Receiver,
			TokenAddress:       bridge.DefaultNativeTokenAddress,
			DepositAmount:      eventBody.Amount,
			Block:              int64(log.BlockNumber),
			SourceAddress:      from.String(),
			ReferralId:         eventBody.ReferralId,
		}
	case EventV1DepositedERC20:
		eventBody := new(v1.BridgeDepositedERC20)
		if err = p.abiV1.UnpackIntoInterface(eventBody, EventToEventName[eventType], log.Data); err != nil {
			return nil, bridgeTypes.ErrFailedUnpackLogs
		}
		unpackedData = &db.DepositData{
			DepositIdentifier:  id,
			DestinationChainId: eventBody.Network,
			DestinationAddress: eventBody.Receiver,
			DepositAmount:      eventBody.Amount,
			TokenAddress:       strings.ToLower(eventBody.Token.String()),
			Block:              int64(log.BlockNumber),
			SourceAddress:      from.String(),
			ReferralId:         0, // v1 does not have referralId
		}
	case EventV2DepositedERC20:
		eventBody := new(v2.BridgeDepositedERC20)
		if err = p.abiV2.UnpackIntoInterface(eventBody, EventToEventName[eventType], log.Data); err != nil {
			return nil, bridgeTypes.ErrFailedUnpackLogs
		}
		unpackedData = &db.DepositData{
			DepositIdentifier:  id,
			DestinationChainId: eventBody.Network,
			DestinationAddress: eventBody.Receiver,
			DepositAmount:      eventBody.Amount,
			TokenAddress:       strings.ToLower(eventBody.Token.String()),
			Block:              int64(log.BlockNumber),
			SourceAddress:      from.String(),
			ReferralId:         eventBody.ReferralId,
		}
	default:
		return nil, bridgeTypes.ErrUnsupportedEvent
	}

	return unpackedData, nil
}

func (p *Client) validateConfirmations(receipt *types.Receipt) error {
	curHeight, err := p.GetCurrentBlockNumber()
	if err != nil {
		return errors.Wrap(err, "failed to get current block number")
	}

	// including the current block
	if receipt.BlockNumber.Uint64()+p.chain.Confirmations-1 > curHeight {
		return bridgeTypes.ErrTxNotConfirmed
	}

	return nil
}

func (p *Client) GetCurrentBlockNumber() (uint64, error) {
	val, err, _ := p.reqGroup.Do("getCurrentBlockNumber", func() (interface{}, error) {
		return p.chain.Rpc.BlockNumber(context.Background())
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to get current block number")
	}

	return val.(uint64), nil
}

package solana

import (
	"context"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/solana/contract"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
	"math/big"
)

func (p *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	signature, err := solana.SignatureFromBase58(id.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tx signature")
	}

	out, err := p.chain.Rpc.GetTransaction(context.Background(), signature, &rpc.GetTransactionOpts{
		Encoding: solana.EncodingBase64,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction")
	}

	if out.Meta.Err != nil {
		return nil, bridgeTypes.ErrTxFailed
	}

	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(out.Transaction.GetBinary()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode transaction")
	}

	instructions, err := contract.DecodeInstructions(&tx.Message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode instructions")
	}

	if len(instructions) <= id.TxNonce {
		return nil, bridgeTypes.ErrInvalidTxNonce
	}
	instr := instructions[id.TxNonce]
	if instr.ProgramID() != contract.ProgramID {
		return nil, bridgeTypes.ErrUnsupportedContract
	}

	switch deposit := instr.Impl.(type) {
	case *contract.DepositNativeInstruction:
		if *deposit.BridgeId != p.chain.BridgeId {
			return nil, bridgeTypes.ErrInvalidReceiverAddress
		}
		return &db.DepositData{
			DepositIdentifier:  id,
			Block:              int64(out.Slot),
			SourceAddress:      deposit.GetSenderAccount().PublicKey.String(),
			DepositAmount:      big.NewInt(int64(*deposit.Amount)),
			TokenAddress:       bridge.DefaultNativeTokenAddress,
			DestinationAddress: *deposit.Address,
			DestinationChainId: *deposit.ChainId,
		}, nil

	case *contract.DepositSplInstruction:
		if *deposit.BridgeId != p.chain.BridgeId {
			return nil, bridgeTypes.ErrInvalidReceiverAddress
		}
		return &db.DepositData{
			DepositIdentifier:  id,
			Block:              int64(out.Slot),
			SourceAddress:      deposit.GetSenderAccount().PublicKey.String(),
			DepositAmount:      big.NewInt(int64(*deposit.Amount)),
			TokenAddress:       deposit.GetMintAccount().PublicKey.String(),
			DestinationAddress: *deposit.Address,
			DestinationChainId: *deposit.ChainId,
		}, nil

	case *contract.DepositWrappedInstruction:
		if *deposit.BridgeId != p.chain.BridgeId {
			return nil, bridgeTypes.ErrInvalidReceiverAddress
		}
		return &db.DepositData{
			DepositIdentifier:  id,
			Block:              int64(out.Slot),
			SourceAddress:      deposit.GetSenderAccount().PublicKey.String(),
			DepositAmount:      big.NewInt(int64(*deposit.Amount)),
			TokenAddress:       deposit.GetMintAccount().PublicKey.String(),
			DestinationAddress: *deposit.Address,
			DestinationChainId: *deposit.ChainId,
		}, nil
	}

	return nil, bridgeTypes.ErrDepositNotFound
}

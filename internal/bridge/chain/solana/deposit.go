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

	var depositType, bridgeId, chainId, address string
	var amount uint64
	var sender solana.PublicKey
	token := bridge.DefaultNativeTokenAddress

	for _, instr := range instructions {
		if instr.ProgramID() != contract.ProgramID {
			continue
		}
		depositNative, ok := instr.Impl.(*contract.DepositNativeInstruction)
		if ok {
			depositType = DepositedNative
			bridgeId = *depositNative.BridgeId
			amount = *depositNative.Amount
			chainId = *depositNative.ChainId
			address = *depositNative.Address
			sender = depositNative.GetSenderAccount().PublicKey
			break
		}
		depositSpl, ok := instr.Impl.(*contract.DepositSplInstruction)
		if ok {
			depositType = DepositedSPL
			bridgeId = *depositSpl.BridgeId
			amount = *depositSpl.Amount
			chainId = *depositNative.ChainId
			address = *depositNative.Address
			sender = depositSpl.GetSenderAccount().PublicKey
			token = depositSpl.GetMintAccount().PublicKey.String()
			break
		}
		depositWrapped, ok := instr.Impl.(*contract.DepositWrappedInstruction)
		if ok {
			depositType = DepositedWrapped
			bridgeId = *depositWrapped.BridgeId
			amount = *depositWrapped.Amount
			chainId = *depositNative.ChainId
			address = *depositNative.Address
			sender = depositWrapped.GetSenderAccount().PublicKey
			token = depositWrapped.GetMintAccount().PublicKey.String()
			break
		}
	}

	if depositType == "" {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	if bridgeId != p.chain.BridgeId {
		return nil, bridgeTypes.ErrInvalidReceiverAddress
	}

	return &db.DepositData{
		DepositIdentifier:  id,
		Block:              int64(out.Slot),
		SourceAddress:      sender.String(),
		DepositAmount:      big.NewInt(int64(amount)),
		TokenAddress:       token,
		DestinationAddress: address,
		DestinationChainId: chainId,
	}, nil
}

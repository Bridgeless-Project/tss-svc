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
	if id.TxNonce != 0 {
		return nil, bridgeTypes.ErrInvalidTxNonce
	}
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

	var depositType, bridgeId, chainId, address, sender string
	var amount uint64
	token := bridge.DefaultNativeTokenAddress

	for _, instr := range instructions {
		if instr.ProgramID() != contract.ProgramID {
			continue
		}
		switch deposit := instr.Impl.(type) {
		case *contract.DepositNativeInstruction:
			depositType = DepositedNative
			bridgeId, amount, chainId, address = *deposit.BridgeId, *deposit.Amount, *deposit.ChainId, *deposit.Address
			sender = deposit.GetSenderAccount().PublicKey.String()
			token = bridge.DefaultNativeTokenAddress
			break

		case *contract.DepositSplInstruction:
			depositType = DepositedSPL
			bridgeId, amount, chainId, address = *deposit.BridgeId, *deposit.Amount, *deposit.ChainId, *deposit.Address
			sender = deposit.GetSenderAccount().PublicKey.String()
			token = deposit.GetMintAccount().PublicKey.String()
			break

		case *contract.DepositWrappedInstruction:
			depositType = DepositedWrapped
			bridgeId, amount, chainId, address = *deposit.BridgeId, *deposit.Amount, *deposit.ChainId, *deposit.Address
			sender = deposit.GetSenderAccount().PublicKey.String()
			token = deposit.GetMintAccount().PublicKey.String()
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
		SourceAddress:      sender,
		DepositAmount:      big.NewInt(int64(amount)),
		TokenAddress:       token,
		DestinationAddress: address,
		DestinationChainId: chainId,
	}, nil
}

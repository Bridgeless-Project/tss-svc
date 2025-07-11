package ton

import (
	"context"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

func (c *Client) getWithdrawalJettonHash(deposit db.Deposit) ([]byte, error) {
	master, err := c.Client.GetMasterchainInfo(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the master chain info")
	}

	networkCell, err := getNetworkCell(deposit.WithdrawalChainId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the network cell")
	}

	receiverCell, err := getAddressCell(deposit.Receiver)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get receiver cell")
	}

	var wrappedBit int64
	if deposit.IsWrappedToken {
		wrappedBit = trueBit
	}

	withdrawalTokenCell, err := getAddressCell(deposit.WithdrawalToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get withdrawal token address cell")
	}

	withdrawalAmount, ok := big.NewInt(0).SetString(deposit.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to parse withdrawal amount")
	}

	txNonce := big.NewInt(0).SetUint64(uint64(deposit.TxNonce))
	txHash := big.NewInt(0).SetBytes(hexutil.MustDecode(deposit.TxHash))
	res, err := c.Client.RunGetMethod(context.Background(),
		master,
		c.BridgeContractAddress,
		withdrawalJettonHashMethod,
		withdrawalAmount,
		receiverCell.BeginParse(),
		txHash,
		txNonce,
		networkCell.BeginParse(),
		wrappedBit,
		withdrawalTokenCell.BeginParse(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the jetton hash")
	}

	resBig, err := res.Int(0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the jetton hash")
	}

	return resBig.Bytes(), nil
}

package ton

import (
	"context"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
)

func (c *Client) getWithdrawalNativeHash(deposit db.Deposit) ([]byte, error) {
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
		return nil, errors.Wrap(err, "failed to get receiver address cell")
	}

	withdrawalAmount, ok := big.NewInt(0).SetString(deposit.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to parse withdrawal amount")
	}

	txHash := big.NewInt(0).SetBytes(TxHashToBytes32(deposit.TxHash))
	txNonce := big.NewInt(0).SetUint64(uint64(deposit.TxNonce))
	res, err := c.Client.RunGetMethod(
		context.Background(),
		master,
		c.BridgeContractAddress,
		withdrawalNativeHashMethod,
		withdrawalAmount,
		receiverCell.BeginParse(),
		txHash,
		txNonce,
		networkCell.BeginParse(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the native hash")
	}

	resBig, err := res.Int(0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the withdrawal native hash")
	}

	return resBig.Bytes(), nil
}

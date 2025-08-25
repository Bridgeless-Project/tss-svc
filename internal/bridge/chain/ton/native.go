package ton

import (
	"context"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
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

	_, _, err = base58.CheckDecode(deposit.TxHash)
	if err == nil {
		h := sha3.New256()

		_, err = h.Write([]byte(deposit.TxHash))
		if err != nil {
			return nil, errors.Wrap(err, "failed to hash the tx hash")
		}
		deposit.TxHash = string(h.Sum(nil))
	}

	hashBytes, err := hexutil.Decode(deposit.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode hash")
	}

	txHash := big.NewInt(0).SetBytes(hashBytes)
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

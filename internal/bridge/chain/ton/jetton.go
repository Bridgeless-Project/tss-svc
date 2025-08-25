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
	_, _, err = base58.CheckDecode(deposit.TxHash)
	if err == nil {
		h := sha3.New256()

		_, err = h.Write([]byte(deposit.TxHash))
		if err != nil {
			return nil, errors.Wrap(err, "failed to hash the tx hash")
		}
		deposit.TxHash = string(h.Sum(nil))
	}

	txhash, err := hexutil.Decode(deposit.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode hash")
	}

	txHash := big.NewInt(0).SetBytes(txhash)
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

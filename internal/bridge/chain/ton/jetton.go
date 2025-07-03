package ton

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

func (c *Client) parseDepositJettonBody(body *cell.Cell) (*depositJettonContent, error) {
	slice := body.BeginParse()

	slice.MustLoadInt(32)

	sender, err := slice.LoadAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error loading sender")
	}

	amount, err := slice.LoadInt(intBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error loading amount")
	}

	isWrapped, err := slice.LoadBoolBit()
	if err != nil {
		return nil, errors.Wrap(err, "error loading wrapped")
	}

	tokenAddr, err := slice.LoadAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error loading amount")
	}

	receiverCell, err := body.PeekRef(0)
	if err != nil {
		return nil, errors.Wrap(err, "error getting address")
	}

	receiver, err := receiverCell.BeginParse().LoadSlice(receiverBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing receiver")
	}

	networkCell, err := body.PeekRef(1)
	if err != nil {
		return nil, errors.Wrap(err, "error getting network")
	}

	network, err := networkCell.BeginParse().LoadStringSnake()
	if err != nil {
		return nil, errors.Wrap(err, "error loading network")
	}

	return &depositJettonContent{
		Sender:       sender.Testnet(c.RPC.IsTestnet),
		Amount:       big.NewInt(amount),
		Receiver:     cleanPrintable(string(receiver)),
		ChainId:      cleanPrintable(network),
		IsWrapped:    isWrapped,
		TokenAddress: tokenAddr.Testnet(c.RPC.IsTestnet),
	}, nil
}

func (c *Client) getWithdrawalJettonHash(deposit db.Deposit) ([]byte, error) {
	master, err := c.Client.GetMasterchainInfo(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the master chain info")
	}

	networkCell, err := getNetworkCell(deposit.WithdrawalChainId)

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

	res, err := c.Client.RunGetMethod(context.Background(), master, c.BridgeContractAddress, withdrawalJettonHashMethod, withdrawalAmount,
		receiverCell.BeginParse(), big.NewInt(0).SetBytes(hexutil.MustDecode(deposit.TxHash)), big.NewInt(int64(deposit.TxNonce)),
		networkCell.BeginParse(), wrappedBit, withdrawalTokenCell.BeginParse())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the native hash")
	}

	resBig, err := res.Int(0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the native hash")
	}

	return resBig.Bytes(), nil
}

func formJettonDepositData(content *depositJettonContent, tx *tlb.Transaction) *db.DepositData {
	return &db.DepositData{
		Block:              int64(tx.LT),
		SourceAddress:      content.Sender.String(),
		DepositAmount:      content.Amount,
		TokenAddress:       content.TokenAddress.String(),
		DestinationChainId: content.ChainId,
		DestinationAddress: content.Receiver,
	}
}

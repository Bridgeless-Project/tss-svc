package ton

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
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

	receiver, err := receiverCell.BeginParse().LoadBigUInt(receiverBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing receiver")
	}

	// TODO: Add addresses definition fro different chains

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
		Receiver:     hexutil.Encode(receiver.Bytes()),
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

	networkCell := cell.BeginCell()
	if err = networkCell.StoreStringSnake(deposit.WithdrawalChainId); err != nil {
		return nil, errors.Wrap(err, "failed to store network")
	}

	addrCell := cell.BeginCell()
	receiverAddr, err := address.ParseAddr(deposit.Receiver)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse receiver address")
	}
	if err = addrCell.StoreAddr(receiverAddr); err != nil {
		return nil, errors.Wrap(err, "failed to store receiver")
	}

	var wrappedBit int64
	if deposit.IsWrappedToken {
		wrappedBit = trueBit
	}

	tokenAddrCell := cell.BeginCell()
	withdrawalTokenAddr, err := address.ParseAddr(deposit.WithdrawalToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse withdrawal token address")
	}
	if err = tokenAddrCell.StoreAddr(withdrawalTokenAddr); err != nil {
		return nil, errors.Wrap(err, "failed to store receiver")
	}

	withdrawalAmount, ok := big.NewInt(0).SetString(deposit.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to parse withdrawal amount")
	}

	res, err := c.Client.RunGetMethod(context.Background(), master, c.BridgeContractAddress, withdrawalJettonHashMethod, withdrawalAmount,
		addrCell.ToSlice(), big.NewInt(0).SetBytes(hexutil.MustDecode(deposit.TxHash)), big.NewInt(deposit.DepositBlock),
		networkCell.ToSlice(), wrappedBit, tokenAddrCell.ToSlice())
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
	}
}

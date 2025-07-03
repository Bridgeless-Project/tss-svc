package ton

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

func (c *Client) parseDepositNativeBody(body *cell.Cell) (*depositNativeContent, error) {
	slice := body.BeginParse()
	slice.MustLoadInt(opCodeBitSize)

	sender, err := slice.LoadAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing sender address")
	}

	amount, err := slice.LoadInt(intBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing amount")
	}

	receiverCell, err := body.PeekRef(0)
	if err != nil {
		return nil, errors.Wrap(err, "error getting receiver ref")
	}

	receiver, err := receiverCell.BeginParse().LoadSlice(receiverBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing receiver address")
	}

	networkCell, err := body.PeekRef(1)
	if err != nil {
		return nil, errors.Wrap(err, "error getting network ref")
	}

	network, err := networkCell.BeginParse().LoadStringSnake()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing network")
	}

	return &depositNativeContent{
		Sender:   sender.Testnet(c.RPC.IsTestnet),
		Amount:   big.NewInt(amount),
		Receiver: cleanPrintable(string(receiver)),
		ChainId:  cleanPrintable(network),
	}, nil

}

func (c *Client) getWithdrawalNativeHash(deposit db.Deposit) ([]byte, error) {
	master, err := c.Client.GetMasterchainInfo(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the master chain info")
	}

	addrSlice := cell.BeginCell()
	receiverAddr, err := address.ParseAddr(deposit.Receiver)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse receiver address")
	}
	if err = addrSlice.StoreAddr(receiverAddr.Testnet(true)); err != nil {
		return nil, errors.Wrap(err, "failed to store receiver")
	}

	withdrawalAmount, ok := big.NewInt(0).SetString(deposit.WithdrawalAmount, 10)
	if !ok {
		return nil, errors.New("failed to parse withdrawal amount")
	}

	hashBytes, err := hexutil.Decode(deposit.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode hash")
	}

	networkSlc := cell.BeginCell()
	fmt.Println("TxHash: ", deposit.TxHash)
	fmt.Println("WithdrawalAmount: ", withdrawalAmount.String())
	fmt.Println("Receiver: ", receiverAddr.String())
	fmt.Println("Network: ", deposit.WithdrawalChainId)
	fmt.Println("TxNonce: ", deposit.TxNonce)
	fmt.Println("Network: ", deposit.WithdrawalChainId)
	fmt.Println("network bytes: ", []byte(deposit.WithdrawalChainId))
	fmt.Println("len network: ", len([]byte(deposit.WithdrawalChainId)))

	networkBytes, err := fillBytesToSize(deposit.WithdrawalChainId, 32, 0x00)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fill bytes to size")
	}
	if err = networkSlc.StoreSlice(networkBytes, 256); err != nil {
		return nil, errors.Wrap(err, "failed to store network")
	}

	fmt.Println("network slice: ", networkSlc)

	res, err := c.Client.RunGetMethod(context.Background(), master,
		c.BridgeContractAddress, withdrawalNativeHashMethod, withdrawalAmount,
		addrSlice.EndCell().BeginParse(), big.NewInt(0).SetBytes(hashBytes),
		big.NewInt(int64(deposit.TxNonce)), networkSlc.EndCell().BeginParse())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the native hash")
	}

	resBig, err := res.Int(0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the withdrawal native hash")
	}

	return resBig.Bytes(), nil
}

func formNativeDepositData(content *depositNativeContent, tx *tlb.Transaction) *db.DepositData {
	return &db.DepositData{
		Block:              int64(tx.LT),
		SourceAddress:      content.Sender.String(),
		DepositAmount:      content.Amount,
		TokenAddress:       bridge.DefaultNativeTokenAddress,
		DestinationAddress: content.Receiver,
		DestinationChainId: content.ChainId,
	}
}

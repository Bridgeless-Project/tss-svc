package ton

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/db"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

func parseMsgOpCode(msg *cell.Slice) (string, error) {
	op, err := msg.LoadBigUInt(32)
	if err != nil {
		return "", errors.Wrap(err, "failed to load message opcode")
	}

	return hexutil.EncodeBig(op), nil
}

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

	receiver, err := receiverCell.BeginParse().LoadAddr()
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
		Sender:   sender.String(),
		Amount:   big.NewInt(amount),
		Receiver: receiver.String(),
		Network:  network,
	}, nil

}

func (c *Client) parseDepositJettonBody(body *cell.Cell) (*depositJettonContent, error) {
	slice := body.BeginParse()
	slice.MustLoadInt(opCodeBitSize)

	sender, err := slice.LoadAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error loading sender")
	}

	amount, err := slice.LoadInt(intBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error loading amount")
	}

	wrappedCell, err := body.PeekRef(0)
	if err != nil {
		return nil, errors.Wrap(err, "error getting address")
	}

	isWrapped, err := wrappedCell.BeginParse().LoadBoolBit()
	if err != nil {
		return nil, errors.Wrap(err, "error loading wrapped")
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
		Sender:    sender.String(),
		Amount:    big.NewInt(amount),
		Receiver:  sender.String(),
		Network:   network,
		IsWrapped: isWrapped,
	}, nil
}

func formNativeDepositData(content *depositNativeContent, tx *tlb.Transaction) *db.DepositData {
	return &db.DepositData{
		Block:              int64(tx.LT),
		SourceAddress:      content.Sender,
		DepositAmount:      content.Amount,
		TokenAddress:       bridge.DefaultNativeTokenAddress,
		DestinationAddress: content.Receiver,
		DestinationChainId: content.Network,
	}
}

// func formJettonDepositData(content *depositJettonContent, tx *tlb.Transaction) *db.DepositData {
// 	return &db.DepositData{
// 		Block:              int64(tx.LT),
// 		SourceAddress:      content.Sender.String(),
// 		DepositAmount:      content.Amount,
// 		TokenAddress: content.
// 	}
// }

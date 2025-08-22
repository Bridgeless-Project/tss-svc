package ton

import (
	"fmt"
	"math/big"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	bridgeTypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func (c *Client) GetDepositData(id db.DepositIdentifier) (*db.DepositData, error) {
	tx, err := c.getTxByLtHash(uint64(id.TxNonce), id.TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx")
	}

	fmt.Println("Transaction found:", tx)
	data, err := c.parseDepositData(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse deposit data")
	}

	fmt.Println("data: ", data)

	data.TxHash = id.TxHash
	data.DepositIdentifier = id

	return data, nil
}

// DepositDecoder is a struct that decodes deposit messages from the TON blockchain.
// It implements all the methods required to parse deposit messages and extract relevant data.
type DepositDecoder struct {
	bridgeAddress address.Address
	isTestnet     bool
}

func NewDepositDecoder(bridgeAddress address.Address, isTestnet bool) *DepositDecoder {
	return &DepositDecoder{
		bridgeAddress: bridgeAddress,
		isTestnet:     isTestnet,
	}
}

// parseDepositData parses the deposit data from the transaction and returns the *db.DepositData object or an error.
func (dd DepositDecoder) parseDepositData(tx *tlb.Transaction) (*db.DepositData, error) {
	fmt.Println("parseDepositData")
	if tx.OutMsgCount == 0 {
		return nil, bridgeTypes.ErrDepositNotFound
	}

	msgs, err := tx.IO.Out.ToSlice()
	if err != nil {
		return nil, errors.Wrap(err, "error getting IO out msgs")
	}

	for _, msg := range msgs {
		fmt.Println("MsgType: ", msg.MsgType)
		fmt.Println("Msg: ", msg)

		if msg.MsgType != tlb.MsgTypeExternalOut {
			continue
		}

		opCode, err := parseMsgOpCode(msg.AsExternalOut().Body.BeginParse())
		if err != nil {
			return nil, errors.Wrap(err, errParseExternalMessage.Error())
		}

		fmt.Println("Opcode: ", opCode)
		switch opCode {
		case depositNativeOpCode:
			content, err := dd.parseDepositNativeBody(msg.AsExternalOut().Body)
			if err != nil {
				return nil, errors.Wrap(err, bridgeTypes.ErrDepositNotFound.Error())
			}

			fmt.Println(content)
			return dd.formNativeDepositData(content, tx), nil

		case depositJettonOpCode:
			content, err := dd.parseDepositJettonBody(msg.AsExternalOut().Body)
			if err != nil {
				return nil, errors.Wrap(err, bridgeTypes.ErrDepositNotFound.Error())
			}
			fmt.Println(content)
			return dd.formJettonDepositData(content, tx), nil
		default:
			break
		}
	}

	return nil, bridgeTypes.ErrDepositNotFound
}

// parseDepositJettonBody parses the body of a native deposit message and returns the content and error
func (dd DepositDecoder) parseDepositJettonBody(body *cell.Cell) (*depositJettonContent, error) {
	slice := body.BeginParse()

	// Skip opCode bytes
	_, err := slice.LoadInt(opCodeBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to skip opCode bytes")
	}

	sender, err := slice.LoadAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error loading sender")
	}

	amount, err := slice.LoadInt(amountBitSize)
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

	receiverCell, err := body.PeekRef(receiverCellId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting address")
	}

	receiver, err := receiverCell.BeginParse().LoadSlice(receiverBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing receiver")
	}

	networkCell, err := body.PeekRef(networkCellId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting network")
	}

	network, err := networkCell.BeginParse().LoadStringSnake()
	if err != nil {
		return nil, errors.Wrap(err, "error loading network")
	}

	fmt.Println("Sender: ", sender.Testnet(dd.isTestnet))
	fmt.Println("Amount: ", big.NewInt(amount))
	fmt.Println("Receiver", cleanPrintable(string(receiver)))
	fmt.Println("Network: ", cleanPrintable(network))
	fmt.Println("TokenAddress: ", tokenAddr.Testnet(dd.isTestnet))

	return &depositJettonContent{
		Sender:       sender.Testnet(dd.isTestnet),
		Amount:       big.NewInt(amount),
		Receiver:     cleanPrintable(string(receiver)),
		ChainId:      cleanPrintable(network),
		IsWrapped:    isWrapped,
		TokenAddress: tokenAddr.Testnet(dd.isTestnet),
	}, nil
}

// parseDepositNativeBody parses the body of a native deposit message and returns the content and error
func (dd DepositDecoder) parseDepositNativeBody(body *cell.Cell) (*depositNativeContent, error) {
	slice := body.BeginParse()

	// Skip opCode bytes
	_, err := slice.LoadInt(opCodeBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to skip opCode bytes")
	}

	sender, err := slice.LoadAddr()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing sender address")
	}

	amount, err := slice.LoadInt(amountBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing amount")
	}

	receiverCell, err := body.PeekRef(receiverCellId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting receiver ref")
	}

	receiver, err := receiverCell.BeginParse().LoadSlice(receiverBitSize)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing receiver address")
	}

	networkCell, err := body.PeekRef(networkCellId)
	if err != nil {
		return nil, errors.Wrap(err, "error getting network ref")
	}

	network, err := networkCell.BeginParse().LoadStringSnake()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing network")
	}

	return &depositNativeContent{
		Sender:   sender.Testnet(dd.isTestnet),
		Amount:   big.NewInt(amount),
		Receiver: cleanPrintable(string(receiver)),
		ChainId:  cleanPrintable(network),
	}, nil
}

// formJettonDepositData creates a DepositData object from the parsed depositJettonContent and transaction.
func (dd DepositDecoder) formJettonDepositData(content *depositJettonContent, tx *tlb.Transaction) *db.DepositData {

	fmt.Println("Forming jetton deposit data with content:", content, "and transaction:", tx)
	fmt.Println("Content Sender:", content.Sender.String())
	fmt.Println("Content Receiver:", cleanPrintable(content.Receiver))
	fmt.Println("Content Amount:", content.Amount)
	fmt.Println("Content ChainId:", content.ChainId)
	fmt.Println("Content TokenAddress:", content.TokenAddress.String())
	fmt.Println("Content IsWrapped:", content.IsWrapped)
	fmt.Println("Transaction LT:", tx.LT)
	fmt.Println("Transaction Hash:", tx.Hash)
	fmt.Println("DestinationChainId: ", content.ChainId)
	fmt.Println("DestinationAddress: ", content.Receiver)

	return &db.DepositData{
		Block:              int64(tx.LT),
		SourceAddress:      content.Sender.String(),
		DepositAmount:      content.Amount,
		TokenAddress:       content.TokenAddress.String(),
		DestinationChainId: content.ChainId,
		DestinationAddress: content.Receiver,
	}
}

// formNativeDepositData creates a DepositData object from the parsed depositNativeContent and transaction.
func (dd DepositDecoder) formNativeDepositData(content *depositNativeContent, tx *tlb.Transaction) *db.DepositData {

	fmt.Println("Forming native deposit data with content:", content, "and transaction:", tx)
	fmt.Println("Content Sender:", content.Sender.String())
	fmt.Println("Content Amount:", content.Amount)
	fmt.Println("Content Receiver:", content.Receiver)
	fmt.Println("Content ChainId:", content.ChainId)

	return &db.DepositData{
		Block:              int64(tx.LT),
		SourceAddress:      content.Sender.String(),
		DepositAmount:      content.Amount,
		TokenAddress:       bridge.DefaultNativeTokenAddress,
		DestinationAddress: content.Receiver,
		DestinationChainId: content.ChainId,
	}
}

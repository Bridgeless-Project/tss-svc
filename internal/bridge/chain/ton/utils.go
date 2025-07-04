package ton

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	tonAddress "github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"strings"
	"unicode"
)

func parseMsgOpCode(msg *cell.Slice) (string, error) {
	op, err := msg.LoadBigUInt(opCodeBitSize)
	if err != nil {
		return "", errors.Wrap(err, "failed to load message opcode")
	}

	return hexutil.Encode(op.Bytes()), nil
}

func cleanPrintable(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func getAddressCell(addr string) (*cell.Cell, error) {
	address, err := tonAddress.ParseAddr(addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse address")
	}

	addressCell := cell.BeginCell()
	if err = addressCell.StoreAddr(address); err != nil {
		return nil, errors.Wrap(err, "failed to store address")
	}

	return addressCell.EndCell(), nil
}

func getNetworkCell(network string) (*cell.Cell, error) {
	networkCell := cell.BeginCell()
	networkBytes, err := fillBytesToSize(network, networkCellSizeBytes, 0x00)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fill bytes")
	}

	if err = networkCell.StoreSlice(networkBytes, networkCellSizeBit); err != nil {
		return nil, errors.Wrap(err, "failed to store bytes")
	}

	return networkCell.EndCell(), nil
}

func fillBytesToSize(str string, size int, fill byte) ([]byte, error) {
	if size == 0 {
		size = 32
	}
	if fill == 0 {
		fill = 0x00
	}
	raw := []byte(str)
	if len(raw) > size {
		return nil, errors.New(fmt.Sprintf("\"%s\" is longer than %d bytes", str, size))
	}

	buf := bytes.Repeat([]byte{fill}, size)
	copy(buf, raw)

	return buf, nil
}

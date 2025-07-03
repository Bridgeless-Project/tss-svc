package ton

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"strings"
	"unicode"
)

func parseMsgOpCode(msg *cell.Slice) (string, error) {
	op, err := msg.LoadBigUInt(32)
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

	fmt.Println("buffer: ", buf)
	fmt.Println("len buf: ", len(buf))
	return buf, nil
}

package ton

import (
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

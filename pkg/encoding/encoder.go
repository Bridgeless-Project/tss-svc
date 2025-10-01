package encoding

import (
	"encoding/base64"
	"encoding/hex"
	"math"

	"github.com/btcsuite/btcd/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type Type byte

const (
	TypeUTF8        Type = 0x01
	TypeHex         Type = 0x02
	TypeHexCheckSum Type = 0x03
	TypeBase58      Type = 0x04
	TypeBase64      Type = 0x05
	TypeBase64Url   Type = 0x06
)

type Encoder interface {
	Encode(raw []byte) string
}

func GetEncoder(t Type) Encoder {
	switch t {
	case TypeUTF8:
		return &UTF8{}
	case TypeHex:
		return &Hex{}
	case TypeHexCheckSum:
		return &HexCheckSum{}
	case TypeBase58:
		return &Base58{}
	case TypeBase64:
		return &Base64{}
	case TypeBase64Url:
		return &Base64Url{}
	default:
		return nil
	}
}

type UTF8 struct{}

func (d *UTF8) Encode(raw []byte) string {
	return string(raw)
}

type Hex struct{}

func (d *Hex) Encode(raw []byte) string {
	return "0x" + hex.EncodeToString(raw)
}

type Base58 struct{}

func (d *Base58) Encode(raw []byte) string {
	return base58.Encode(raw)
}

type Base64 struct{}

func (d *Base64) Encode(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

type Base64Url struct{}

func (d *Base64Url) Encode(raw []byte) string {
	return base64.URLEncoding.EncodeToString(raw)
}

// HexCheckSum encodes to hex with EIP-55 checksum.
// If the value is longer than 40 characters, the checksum won't be applied to the full length.
type HexCheckSum struct{}

func (d *HexCheckSum) Encode(raw []byte) string {
	buf := make([]byte, 2+2*len(raw))
	copy(buf[:2], "0x")
	hex.Encode(buf[2:], raw)

	sha := sha3.NewLegacyKeccak256()
	sha.Write(buf[2:])
	hash := sha.Sum(nil)

	loopCondition := int(math.Min(42, float64(len(buf))))

	for i := 2; i < loopCondition; i++ {
		hashByte := hash[(i-2)/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if buf[i] > '9' && hashByte > 7 {
			buf[i] -= 32
		}
	}
	return string(buf)
}

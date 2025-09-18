package encoding

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcutil/base58"
)

type Type byte

const (
	TypeUTF8      Type = 0x01
	TypeHex       Type = 0x02
	TypeBase58    Type = 0x03
	TypeBase64    Type = 0x04
	TypeBase64Url Type = 0x05
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

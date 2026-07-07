package client

import (
	"errors"
)

const (
	memoMagicByte        = 0xFF
	memoMagicByteIdx     = 0
	memoReferralIdLength = 2

	memoV3HeaderLength = 4
	memoV3Version      = 0x03

	memoV3VersionIdx     = 1
	memoV3ChunksCountIdx = 2
	memoV3SwapFlagIdx    = 3

	memoV3NoSwap = 0x00
	memoV3Swap   = 0x01
)

type MemoReader struct {
	raw []byte
	pos int
}

func NewMemoReader(raw []byte) *MemoReader {
	return &MemoReader{
		raw: raw,
		pos: 0,
	}
}

func (r *MemoReader) ReadByte() (byte, error) {
	if r.pos >= len(r.raw) {
		return 0, errors.New("no more bytes to read from memo")
	}

	b := r.raw[r.pos]
	r.pos++

	return b, nil
}

func (r *MemoReader) ReadBytes(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.raw) {
		return nil, errors.New("invalid length to read from memo")
	}

	b := r.raw[r.pos : r.pos+n]
	r.pos += n

	return b, nil
}

func (r *MemoReader) ReadLenBytes() ([]byte, error) {
	l, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, errors.New("memo field length cannot be zero")
	}

	return r.ReadBytes(int(l))
}

func (r *MemoReader) Remaining() int {
	return len(r.raw) - r.pos
}

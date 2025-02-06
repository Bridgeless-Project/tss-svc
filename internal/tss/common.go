package tss

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/hyle-team/tss-svc/internal/core"
)

const (
	OutChannelSize = 1000
	EndChannelSize = 1
	MsgsCapacity   = 100
)

type partyMsg struct {
	Sender      core.Address
	WireMsg     []byte
	IsBroadcast bool
}

func Verify(localParty *keygen.LocalPartySaveData, inputData []byte, signature *common.SignatureData) bool {
	pk := ecdsa.PublicKey{
		Curve: tss.EC(),
		X:     localParty.ECDSAPub.X(),
		Y:     localParty.ECDSAPub.Y(),
	}
	data := big.NewInt(0).SetBytes(inputData)

	return ecdsa.Verify(&pk, data.Bytes(), new(big.Int).SetBytes(signature.R), new(big.Int).SetBytes(signature.S))
}

type SessionParams struct {
	Id        int64     `fig:"session_id,required"`
	StartTime time.Time `fig:"start_time,required"`
	Threshold int       `fig:"threshold,required"`
}

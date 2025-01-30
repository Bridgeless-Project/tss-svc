package types

import (
	coretypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/pkg/errors"
)

var (
	ErrPairNotFound                = errors.New("pair not found")
	ErrTokenInfoNotFound           = errors.New("token info not found")
	ErrTransactionAlreadySubmitted = errors.New("transaction already submitted")
)

type TokenPairer interface {
	GetDestinationTokenInfo(
		srcChainId string,
		srcTokenAddr string,
		dstChainId string,
	) (coretypes.TokenInfo, error)
}

type Tokener interface {
	GetTokenInfo(chainId string, addr string) (coretypes.TokenInfo, error)
}

type TxSubmitter interface {
	SubmitDeposits(depositTxs ...coretypes.Transaction) error
}

type Bridger interface {
	Tokener
	TokenPairer
	TxSubmitter
}

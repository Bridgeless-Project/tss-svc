package core

import "github.com/pkg/errors"

var (
	ErrTokenPairNotFound            = errors.New("token pair not found")
	ErrTokenInfoNotFound            = errors.New("token info not found")
	ErrTransactionAlreadySubmitted  = errors.New("transaction already submitted")
	ErrSourceTokenInfoNotFound      = errors.New("source token not found")
	ErrDestinationTokenInfoNotFound = errors.New("destination token not found")
	ErrReferralNotFound             = errors.New("referral not found")

	ErrEpochNotFound = errors.New("epoch not found")
)

func IsInvalidDepositError(err error) bool {
	return errors.Is(err, ErrTokenPairNotFound) ||
		errors.Is(err, ErrTokenInfoNotFound) ||
		errors.Is(err, ErrDestinationTokenInfoNotFound) ||
		errors.Is(err, ErrSourceTokenInfoNotFound) ||
		errors.Is(err, ErrReferralNotFound)
}

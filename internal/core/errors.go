package core

import "github.com/pkg/errors"

var (
	ErrTokenPairNotFound           = errors.New("token pair not found")
	ErrTokenInfoNotFound           = errors.New("token info not found")
	ErrTransactionAlreadySubmitted = errors.New("transaction already submitted")
)

func IsInvalidDepositError(err error) bool {
	return errors.Is(err, ErrTokenPairNotFound) || errors.Is(err, ErrTokenInfoNotFound)
}

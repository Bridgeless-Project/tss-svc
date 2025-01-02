package secrets

import "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"

type Storage interface {
	GetKeygenPreParams() keygen.LocalPreParams
}

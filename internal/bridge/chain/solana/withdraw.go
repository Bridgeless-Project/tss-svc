package solana

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/gagliardetto/solana-go"
	"math/big"
	"strconv"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/db"
	"github.com/pkg/errors"
)

func (p *Client) WithdrawalAmountValid(amount *big.Int) bool {
	// Solana token amounts are uint64, bigger (or negative) numbers are invalid
	if !amount.IsUint64() {
		return false
	}
	return amount.Cmp(bridge.ZeroAmount) == 1
}

func (p *Client) GetSignHash(data db.Deposit) ([]byte, error) {
	amount, err := strconv.ParseUint(data.WithdrawalAmount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse withdrawal amount")
	}
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, amount)
	// todo: reconsider nonce
	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, uint64(data.TxNonce))
	receiver, err := solana.PublicKeyFromBase58(data.Receiver)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse receiver address")
	}

	buffer := []byte("withdraw")
	buffer = append(buffer, []byte(p.chain.BridgeId)...)
	buffer = append(buffer, amountBytes...)
	buffer = append(buffer, nonceBytes...)
	buffer = append(buffer, receiver.Bytes()...)

	if data.WithdrawalToken != bridge.DefaultNativeTokenAddress {
		token, err := solana.PublicKeyFromBase58(data.WithdrawalToken)
		if err != nil {
			return nil, err
		}
		buffer = append(buffer, token.Bytes()...)
	}

	hash := sha256.Sum256(buffer)
	return hash[:], nil
}

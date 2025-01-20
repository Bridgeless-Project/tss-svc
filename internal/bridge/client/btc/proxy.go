package btc

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/hyle-team/tss-svc/internal/bridge"
	"github.com/hyle-team/tss-svc/internal/bridge/chain"
	bridgeTypes "github.com/hyle-team/tss-svc/internal/bridge/types"
	"github.com/hyle-team/tss-svc/internal/db"
	"gitlab.com/distributed_lab/logan/v3"
	"math/big"
)

// minSatoshisPerOutput calculated for P2PKH
const minSatoshisPerOutput = 547

type BridgeProxy interface {
	bridgeTypes.Proxy
	SendBitcoins(map[string]*big.Int) (txHash string, err error)
}

type proxy struct {
	chain  chain.Bitcoin
	logger *logan.Entry
}

func (p *proxy) ConstructWithdrawalTx(data db.Deposit) ([]byte, error) {
	//TODO implement me
	return []byte("bitcoin"), nil
}

func NewBridgeProxy(ch chain.Bitcoin, logger *logan.Entry) BridgeProxy {
	return &proxy{chain: ch, logger: logger}
}

func (*proxy) Type() bridgeTypes.ChainType {
	return bridgeTypes.ChainTypeBitcoin
}

func (p *proxy) AddressValid(addr string) bool {
	_, err := btcutil.DecodeAddress(addr, p.chain.Params)
	return err == nil
}

func (p *proxy) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func (p *proxy) bridgeAddr(addr btcutil.Address) bool {
	for _, receiver := range p.chain.Receivers {
		if addr.String() == receiver.String() {
			return true
		}
	}

	return false
}

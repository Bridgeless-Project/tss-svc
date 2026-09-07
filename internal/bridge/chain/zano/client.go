package zano

import (
	"regexp"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	zanoTypes "github.com/Bridgeless-Project/tss-svc/pkg/zano/types"
	"github.com/pkg/errors"
)

// Zano address patterns: - - - - - - - - - - - - - - - Standard - - - - - - - - Integrated - - - - - - - - Auditable - - - - - - - - Auditable Integrated - - - - - - Gateway - - - - - - - - - Gateway Integrated
var addressPattern = regexp.MustCompile(`^(?:Zx[1-9A-HJ-NP-Za-km-z]{95}|iZ[1-9A-HJ-NP-Za-km-z]{106}|aZx[1-9A-HJ-NP-Za-km-z]{96}|aiZ[1-9A-HJ-NP-Za-km-z]{107}|gwZ[1-9A-HJ-NP-Za-km-z]{54}|gwiZ[1-9A-HJ-NP-Za-km-z]{64})$`)

type Client struct {
	chain Chain
}

func (p *Client) ChainId() string {
	return p.chain.Id
}

func (p *Client) Type() chain.Type {
	return chain.TypeZano
}

func (p *Client) AddressValid(addr string) bool {
	return addressPattern.MatchString(addr)
}

func (p *Client) TransactionHashValid(hash string) bool {
	return bridge.DefaultTransactionHashPattern.MatchString(hash)
}

func NewBridgeClient(chain Chain) *Client {
	return &Client{chain}
}

func (p *Client) HealthCheck() error {
	_, err := p.chain.Client.CurrentHeight()
	if err != nil {
		return errors.Wrap(err, "failed to get current height from zano daemon")
	}

	_, err = p.chain.Client.GetWalletInfo()
	if err != nil {
		return errors.Wrap(err, "failed to get wallet info from zano wallet")
	}

	return nil
}

func (p *Client) IsCentralized() bool {
	return false
}

func (p *Client) GetAssetInfo(address string) (*zanoTypes.AssetDescriptor, error) {
	resp, err := p.chain.Client.GetAssetInfo(address)
	if err != nil {
		return nil, err
	}

	return &resp.AssetDescriptor, nil
}

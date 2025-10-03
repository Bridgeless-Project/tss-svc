package connector

import (
	"context"
	"sync"
	"time"

	coretypes "github.com/Bridgeless-Project/bridgeless-core/v12/types"
	bridgetypes "github.com/Bridgeless-Project/bridgeless-core/v12/x/bridge/types"
	"github.com/Bridgeless-Project/tss-svc/internal/core"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txclient "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const gasLimit = 3_000_000

type Settings struct {
	ChainId     string `fig:"chain_id,required"`
	Denom       string `fig:"denom,required"`
	MinGasPrice uint64 `fig:"min_gas_price,required"`
}

type Connector struct {
	conn *grpc.ClientConn

	transactor txclient.ServiceClient
	txConfiger sdkclient.TxConfig
	auther     authtypes.QueryClient
	querier    bridgetypes.QueryClient

	settings Settings
	account  core.Account

	accountNumber   uint64
	accountSequence uint64
	mu              *sync.Mutex
}

func NewConnector(account core.Account, conn *grpc.ClientConn, settings Settings) (*Connector, error) {
	accountData, err := getAccountData(context.Background(), authtypes.NewQueryClient(conn), account.CosmosAddress())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account data")
	}

	return &Connector{
		conn:       conn,
		transactor: txclient.NewServiceClient(conn),
		txConfiger: authtx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
		auther:     authtypes.NewQueryClient(conn),
		querier:    bridgetypes.NewQueryClient(conn),
		settings:   settings,
		account:    account,

		accountNumber:   accountData.AccountNumber,
		accountSequence: accountData.Sequence,

		mu: &sync.Mutex{},
	}, nil

}

func (c *Connector) HealthCheck() error {
	checker := grpc_health_v1.NewHealthClient(c.conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := checker.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		return errors.Wrap(err, "failed to perform health check")
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return errors.Errorf("service not healthy: %s", resp.Status.String())
	}

	return nil
}

func (c *Connector) getAccountSequence() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	seq := c.accountSequence
	c.accountSequence++

	return seq
}

func (c *Connector) submitMsgs(ctx context.Context, msgs ...sdk.Msg) error {
	if len(msgs) == 0 {
		return nil
	}

	feeAmount := gasLimit * c.settings.MinGasPrice

	tx, err := c.buildTx(gasLimit, feeAmount, msgs...)
	if err != nil {
		return errors.Wrap(err, "failed to build transaction")
	}

	res, err := c.transactor.BroadcastTx(ctx, &txclient.BroadcastTxRequest{
		Mode:    txclient.BroadcastMode_BROADCAST_MODE_BLOCK,
		TxBytes: tx,
	})

	if err != nil {
		return errors.Wrap(err, "failed to broadcast transaction")
	}
	if res.TxResponse.Code != txCodeSuccess {
		return errors.Errorf("transaction failed with code %d", res.TxResponse.Code)
	}

	return nil
}

// buildTx builds a transaction from the given messages.
func (c *Connector) buildTx(gasLimit, feeAmount uint64, msgs ...sdk.Msg) ([]byte, error) {
	txBuilder := c.txConfiger.NewTxBuilder()

	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, errors.Wrap(err, "failed to set messages")
	}

	sequence := c.getAccountSequence()

	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(sdk.Coins{sdk.NewInt64Coin(c.settings.Denom, int64(feeAmount))})

	signMode := c.txConfiger.SignModeHandler().DefaultMode()
	err := txBuilder.SetSignatures(signing.SignatureV2{
		PubKey: c.account.PublicKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: nil,
		},
		Sequence: sequence,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to set signature")
	}

	signerData := authsigning.SignerData{
		ChainID:       c.settings.ChainId,
		AccountNumber: c.accountNumber,
		Sequence:      sequence,
	}

	sig, err := clienttx.SignWithPrivKey(signMode, signerData, txBuilder, c.account.PrivateKey(), c.txConfiger, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign with private key")
	}

	if err = txBuilder.SetSignatures(sig); err != nil {
		return nil, errors.Wrap(err, "failed to set signatures")
	}

	return c.txConfiger.TxEncoder()(txBuilder.GetTx())
}

func getAccountData(ctx context.Context, auther authtypes.QueryClient, address core.Address) (*coretypes.EthAccount, error) {
	resp, err := auther.Account(ctx, &authtypes.QueryAccountRequest{Address: address.String()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account")
	}

	account := coretypes.EthAccount{}
	if err = account.Unmarshal(resp.Account.Value); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account")
	}

	return &account, nil
}

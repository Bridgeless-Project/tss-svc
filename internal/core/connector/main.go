package core

import (
	"context"

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
	coretypes "github.com/hyle-team/bridgeless-core/types"
	bridgetypes "github.com/hyle-team/bridgeless-core/x/bridge/types"
	"github.com/hyle-team/tss-svc/internal/core"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var (
	ErrPairNotFound                = errors.New("pair not found")
	ErrTokenInfoNotFound           = errors.New("token info not found")
	ErrTransactionAlreadySubmitted = errors.New("transaction already submitted")
)

type ConnectorSettings struct {
	Account     core.Account `fig:"account_private_key,required"`
	ChainId     string       `fig:"chain_id,required"`
	Denom       string       `fig:"denom,required"`
	MinGasPrice uint64       `fig:"min_gas_price"`
}

type Connector struct {
	transactor txclient.ServiceClient
	txConfiger sdkclient.TxConfig
	auther     authtypes.QueryClient
	querier    bridgetypes.QueryClient

	settings ConnectorSettings
}

func NewConnector(conn *grpc.ClientConn, settings ConnectorSettings) *Connector {
	return &Connector{
		transactor: txclient.NewServiceClient(conn),
		txConfiger: authtx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
		auther:     authtypes.NewQueryClient(conn),
		querier:    bridgetypes.NewQueryClient(conn),
		settings:   settings,
	}
}

func (c *Connector) submitMsgs(msgs ...sdk.Msg) error {
	if len(msgs) == 0 {
		return nil
	}

	tx, err := c.buildTx(0, 0, msgs...)
	if err != nil {
		return errors.Wrap(err, "failed to build simulation transaction")
	}

	simResp, err := c.transactor.Simulate(context.Background(), &txclient.SimulateRequest{TxBytes: tx})
	if err != nil {
		return errors.Wrap(err, "failed to simulate transaction")
	}

	gasLimit := ApproximateGasLimit(simResp.GasInfo.GasUsed)
	feeAmount := gasLimit * c.settings.MinGasPrice

	tx, err = c.buildTx(gasLimit, feeAmount, msgs...)
	if err != nil {
		return errors.Wrap(err, "failed to build transaction")
	}
	res, err := c.transactor.BroadcastTx(context.Background(), &txclient.BroadcastTxRequest{
		Mode:    txclient.BroadcastMode_BROADCAST_MODE_BLOCK,
		TxBytes: tx,
	})
	if err != nil {
		return errors.Wrap(err, "failed to broadcast transaction")
	}
	if res.TxResponse.Code != txCodeSuccess {
		return errors.Errorf("transaction failed with code %d, info %s", res.TxResponse.Code, res.TxResponse.Info)
	}

	return nil
}

// buildTx builds a transaction from the given messages.
func (c *Connector) buildTx(gasLimit, feeAmount uint64, msgs ...sdk.Msg) ([]byte, error) {
	txBuilder := c.txConfiger.NewTxBuilder()

	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, errors.Wrap(err, "failed to set messages")
	}

	// Get account to set sequence number
	acc, err := c.getAccountData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account")
	}

	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(sdk.Coins{sdk.NewInt64Coin(c.settings.Denom, int64(feeAmount))})

	signMode := c.txConfiger.SignModeHandler().DefaultMode()
	err = txBuilder.SetSignatures(signing.SignatureV2{
		PubKey: c.settings.Account.PublicKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: nil,
		},
		Sequence: acc.Sequence,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to set signature")
	}

	signerData := authsigning.SignerData{
		ChainID:       c.settings.ChainId,
		AccountNumber: acc.AccountNumber,
		Sequence:      acc.Sequence,
	}

	sig, err := clienttx.SignWithPrivKey(signMode, signerData, txBuilder, c.settings.Account.PrivateKey(), c.txConfiger, acc.Sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign with private key")
	}

	if err = txBuilder.SetSignatures(sig); err != nil {
		return nil, errors.Wrap(err, "failed to set signatures")
	}

	return c.txConfiger.TxEncoder()(txBuilder.GetTx())
}

func (c *Connector) getAccountData() (*coretypes.EthAccount, error) {
	resp, err := c.auther.Account(context.Background(), &authtypes.QueryAccountRequest{Address: c.settings.Account.CosmosAddress().String()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account")
	}

	account := coretypes.EthAccount{}
	if err = account.Unmarshal(resp.Account.Value); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal account")
	}

	return &account, nil
}

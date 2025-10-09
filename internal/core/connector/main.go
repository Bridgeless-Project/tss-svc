package connector

import (
	"context"
	"fmt"
	"sync"

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
	coretypes "github.com/hyle-team/bridgeless-core/v12/types"
	bridgetypes "github.com/hyle-team/bridgeless-core/v12/x/bridge/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const gasLimit = 3_000_000

type Settings struct {
	ChainId     string `fig:"chain_id,required"`
	Denom       string `fig:"denom,required"`
	MinGasPrice uint64 `fig:"min_gas_price,required"`
}

type Connector struct {
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

func (c *Connector) getAccountSequence() uint64 {
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
		if res.TxResponse.Code != txCodeWrongSequence {
			return errors.Errorf("transaction failed with code %d", res.TxResponse.Code)
		}

		// resubmit the transaction with the correct sequence in the background
		// without returning an error to the caller
		go func() {
			retryNum, ok := ctx.Value("retryNum").(int)
			if !ok {
				retryNum = 0
			}
			if retryNum >= 5 {
				fmt.Println("max retry attempts reached, giving up")
				return
			}
			retryNum++

			// fetch the latest account sequence
			accountData, err := getAccountData(context.Background(), c.auther, c.account.CosmosAddress())
			if err != nil {
				fmt.Println("failed to get account data")
				return
			}

			c.mu.Lock()
			c.accountSequence = accountData.Sequence
			c.mu.Unlock()

			fmt.Printf("resubmitting transaction, attempt %d\n", retryNum)
			ctx = context.WithValue(context.Background(), "retryNum", retryNum)
			if err = c.submitMsgs(ctx, msgs...); err != nil {
				fmt.Printf("failed to resubmit transaction: %v\n", err)
			}
		}()
	}

	return nil
}

// buildTx builds a transaction from the given messages.
func (c *Connector) buildTx(gasLimit, feeAmount uint64, msgs ...sdk.Msg) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

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

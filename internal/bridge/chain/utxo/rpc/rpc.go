package rpc

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	utxohelper "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/helper/factory"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/utils"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type Settings struct {
	Host     string
	User     string
	Password string

	Chain   types.Chain
	Network types.Network
}

type Client struct {
	c        *rpc.Client
	settings Settings

	helper utxohelper.UtxoHelper
	chain  types.Chain
}

func dialRPC(settings Settings) (*rpc.Client, error) {
	authFn := func(h http.Header) error {
		auth := base64.StdEncoding.EncodeToString([]byte(settings.User + ":" + settings.Password))
		h.Set("Authorization", fmt.Sprintf("Basic %s", auth))
		return nil
	}

	// default to http if no scheme is specified
	if !strings.Contains(settings.Host, "://") {
		settings.Host = "http://" + settings.Host
	}

	return rpc.DialOptions(context.Background(), settings.Host, rpc.WithHTTPAuth(authFn))
}

func NewClient(settings Settings) (*Client, error) {
	c, err := dialRPC(settings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to RPC server")
	}

	return &Client{
		c:        c,
		settings: settings,
		helper:   factory.NewUtxoHelper(settings.Chain, settings.Network),
		chain:    settings.Chain,
	}, nil
}

//
// DAEMON METHODS
//

func (c *Client) EstimateFee() (btcutil.Amount, error) {
	var (
		fee float64
		err error
	)

	switch c.chain {
	case types.ChainBch:
		fee, err = c.estimateFeeBch()
	case types.ChainBtc:
		fee, err = c.estimateFeeBtc()
	default:
		return 0, errors.Errorf("unsupported chain: %s", c.chain)
	}
	if err != nil {
		return 0, errors.Wrap(err, "failed to estimate fee")
	}

	amt, err := btcutil.NewAmount(fee)
	if err != nil {
		return 0, errors.Wrap(err, "failed to convert fee to btcutil.Amount")
	}
	if amt < utils.DefaultFeeRateBtcPerKvb {
		return 0, errors.New("estimated fee is too low")
	}

	return amt, nil // Convert from BTC/BCH to satoshis
}

func (c *Client) estimateFeeBch() (float64, error) {
	var result float64
	err := c.Call(&result, "estimatefee")
	return result, extractRpcError(err)
}

func (c *Client) estimateFeeBtc() (float64, error) {
	const confirmationsTarget = 5
	var result btcjson.EstimateSmartFeeResult
	err := c.Call(&result, "estimatesmartfee", confirmationsTarget, nil)
	if err != nil {
		return 0, extractRpcError(err)
	}
	if result.Errors != nil {
		return 0, errors.Errorf("%v", result.Errors)
	}
	if result.FeeRate == nil {
		return 0, errors.New("fee rate is nil")
	}

	return *result.FeeRate, nil
}

func (c *Client) GetRawTransactionVerbose(txHash string) (*btcjson.TxRawResult, error) {
	var tx btcjson.TxRawResult
	err := c.Call(&tx, "getrawtransaction", txHash, true)
	return &tx, extractRpcError(err)
}

func (c *Client) GetBlockVerbose(hash string) (*btcjson.GetBlockVerboseResult, error) {
	var block btcjson.GetBlockVerboseResult
	err := c.Call(&block, "getblock", hash, 1)
	return &block, extractRpcError(err)
}

func (c *Client) SendRawTransaction(tx *wire.MsgTx) (string, error) {
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return "", errors.Wrap(err, "failed to serialize transaction")
	}

	txHex := hex.EncodeToString(buf.Bytes())
	var maxFee interface{}
	switch c.chain {
	case types.ChainBtc:
		// BTC per kVb; default max fee rate
		maxFee = 0.1
	case types.ChainBch:
		maxFee = false
	}

	var txHash string
	err := c.Call(&txHash, "sendrawtransaction", txHex, maxFee)
	return txHash, extractRpcError(err)
}

func (c *Client) GetBlockCount() (int64, error) {
	var count int64
	err := c.Call(&count, "getblockcount")
	return count, extractRpcError(err)
}

//
// WALLET METHODS
//

func (c *Client) ListUnspent(minConfirm uint64) ([]btcjson.ListUnspentResult, error) {
	var unspent []btcjson.ListUnspentResult
	const maxConfirm = 9999999
	err := c.Call(&unspent, "listunspent", minConfirm, maxConfirm, nil)
	return unspent, extractRpcError(err)
}

func (c *Client) GetWalletInfo() (*btcjson.GetWalletInfoResult, error) {
	var info btcjson.GetWalletInfoResult
	err := c.Call(&info, "getwalletinfo")
	return &info, extractRpcError(err)
}

func (c *Client) LockUnspent(unlock bool, ops []*wire.OutPoint) error {
	var outPoints []btcjson.TransactionInput
	for _, op := range ops {
		outPoints = append(outPoints, btcjson.TransactionInput{
			Txid: op.Hash.String(),
			Vout: op.Index,
		})
	}

	err := c.Call(nil, "lockunspent", unlock, outPoints)
	return extractRpcError(err)
}

func (c *Client) Call(result any, method string, args ...interface{}) error {
	err := c.c.Call(result, method, args...)
	return extractRpcError(err)
}

func (c *Client) InitializeWallet(pubkey *ecdsa.PublicKey, epoch uint32, syncTime time.Time) error {
	walletName := c.getWalletName(epoch, pubkey)

	switch c.chain {
	case types.ChainBch:
		return c.initializeWalletBch(pubkey, walletName)
	case types.ChainBtc:
		return c.initializeWalletBtc(pubkey, walletName, syncTime)
	default:
		return errors.Errorf("unsupported chain: %s", c.chain)
	}
}

func (c *Client) initializeWalletBch(pubkey *ecdsa.PublicKey, walletName string) error {
	if err := c.createWalletBch(walletName); err != nil {
		return errors.Wrap(err, "failed to create wallet")
	}

	newSettings := c.settings
	newSettings.Host = formWalletHost(c.settings.Host, walletName)

	walletClient, err := NewClient(newSettings)
	if err != nil {
		return errors.Wrap(err, "failed to create wallet client")
	}

	if err = walletClient.importAddressBch(c.helper.P2pkhAddress(pubkey)); err != nil {
		return errors.Wrap(err, "failed to import address")
	}

	return nil
}

func (c *Client) createWalletBch(name string) error {
	err := c.Call(nil, "createwallet", name, true, false)
	return extractRpcError(err)
}

func (c *Client) importAddressBch(address string) error {
	err := c.Call(nil, "importaddress", address, "", true, false)
	return extractRpcError(err)
}

func (c *Client) initializeWalletBtc(pubkey *ecdsa.PublicKey, walletName string, syncTime time.Time) error {
	if err := c.createWalletBtc(walletName); err != nil {
		return errors.Wrap(err, "failed to create wallet")
	}

	newSettings := c.settings
	newSettings.Host = formWalletHost(c.settings.Host, walletName)

	walletClient, err := NewClient(newSettings)
	if err != nil {
		return errors.Wrap(err, "failed to create wallet client")
	}

	descriptor := fmt.Sprintf("pkh(%s)", hex.EncodeToString(crypto.CompressPubkey(pubkey)))
	info, err := walletClient.getDescriptorInfo(descriptor)
	if err != nil {
		return errors.Wrap(err, "failed to get descriptor info")
	}

	if err = walletClient.importDescriptors(descriptor, info.Checksum, syncTime); err != nil {
		return errors.Wrap(err, "failed to import descriptors")
	}

	return nil
}

func (c *Client) createWalletBtc(name string) error {
	err := c.Call(nil, "createwallet", name, true, false, "", false, true, true)
	return extractRpcError(err)
}

func (c *Client) importDescriptors(descriptor, checksum string, syncTime time.Time) error {
	fullDescriptor := fmt.Sprintf("%s#%s", descriptor, checksum)
	syncTimestamp := syncTime.Unix()

	request := []ImportDescriptorsRequest{{Descriptor: &fullDescriptor, Time: &syncTimestamp}}

	var resp []ImportDescriptorsResponse
	err := c.Call(&resp, "importdescriptors", request)
	if err != nil {
		return extractRpcError(err)
	}

	for _, result := range resp {
		if !result.Success && result.Error != nil {
			return btcjson.NewRPCError(result.Error.Code, result.Error.Message)
		}
	}

	return nil
}

type ImportDescriptorsRequest struct {
	Descriptor *string `json:"desc,omitempty"`
	Time       *int64  `json:"timestamp,omitempty"`
	Active     *bool   `json:"active,omitempty"`
	Range      []int   `json:"range,omitempty"`
}

type ImportDescriptorsResponse struct {
	Success bool      `json:"success"`
	Error   *RPCError `json:"error,omitempty"`
}

func (c *Client) getDescriptorInfo(descriptor string) (*btcjson.GetDescriptorInfoResult, error) {
	var info btcjson.GetDescriptorInfoResult
	err := c.Call(&info, "getdescriptorinfo", descriptor)
	return &info, extractRpcError(err)
}

// Ethereum RPC returns an error with the response appended to the HTTP status like:
// 404 Not Found: {"error":{"code":-32601,"message":"Method not found"},"id":1}
func extractRpcError(err error) error {
	if err == nil {
		return nil
	}

	// split the error into the HTTP status and the JSON response
	parts := strings.SplitN(err.Error(), ": ", 2)
	if len(parts) != 2 {
		return err
	}

	// parse the JSON response
	var response struct {
		Error RPCError `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(parts[1]), &response); jsonErr != nil {
		return err
	}

	// return the error message
	return btcjson.NewRPCError(response.Error.Code, response.Error.Message)
}

type RPCError struct {
	Code    btcjson.RPCErrorCode `json:"code"`
	Message string               `json:"message"`
}

func formWalletHost(host, wallet string) string {
	if strings.Contains(host, "/wallet") {
		parts := strings.SplitAfter(host, "/wallet")
		return fmt.Sprintf("%s/%s", parts[0], wallet)
	}

	return fmt.Sprintf("%s/wallet/%s", host, wallet)
}

func (c *Client) getWalletName(epoch uint32, pubkey *ecdsa.PublicKey) string {
	return fmt.Sprintf("%v_%s", epoch, c.helper.P2pkhAddress(pubkey))
}

package rpc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	utxotypes "github.com/Bridgeless-Project/tss-svc/internal/bridge/chain/utxo/types"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type Settings struct {
	Host     string
	User     string
	Password string
	Chain    utxotypes.Chain
}

type Client struct {
	c     *rpc.Client
	chain utxotypes.Chain
}

func NewClient(settings Settings) (*Client, error) {
	authFn := func(h http.Header) error {
		auth := base64.StdEncoding.EncodeToString([]byte(settings.User + ":" + settings.Password))
		h.Set("Authorization", fmt.Sprintf("Basic %s", auth))
		return nil
	}

	// default to http if no scheme is specified
	if !strings.Contains(settings.Host, "://") {
		settings.Host = "http://" + settings.Host
	}

	c, err := rpc.DialOptions(context.Background(), settings.Host, rpc.WithHTTPAuth(authFn))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to RPC server")
	}

	return &Client{c, settings.Chain}, nil
}

//
// DAEMON METHODS
//

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
	case utxotypes.ChainBtc:
		// BTC per kVb; default max fee rate
		maxFee = 0.1
	case utxotypes.ChainBch:
		maxFee = false
	}

	var txHash string
	err := c.Call(&txHash, "sendrawtransaction", txHex, maxFee)
	return txHash, extractRpcError(err)
}

//
// WALLET METHODS
//

func (c *Client) FundRawTransaction(
	tx *wire.MsgTx,
	opts btcjson.FundRawTransactionOpts,
) (*btcjson.FundRawTransactionResult, error) {
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return nil, errors.Wrap(err, "failed to serialize transaction")
	}

	txHex := hex.EncodeToString(buf.Bytes())
	args := []interface{}{txHex, opts}
	if c.chain == utxotypes.ChainBtc {
		// optional btc arg for Bitcoin Core
		args = append(args, nil)
	}

	var funded btcjson.FundRawTransactionResult
	err := c.Call(&funded, "fundrawtransaction", args...)
	return &funded, extractRpcError(err)
}

func (c *Client) ListUnspent() ([]btcjson.ListUnspentResult, error) {
	var unspent []btcjson.ListUnspentResult
	const minConfirm = 1
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
		Error struct {
			Code    btcjson.RPCErrorCode `json:"code"`
			Message string               `json:"message"`
		} `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(parts[1]), &response); jsonErr != nil {
		return err
	}

	// return the error message
	return btcjson.NewRPCError(response.Error.Code, response.Error.Message)
}

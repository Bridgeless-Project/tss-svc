# Bridge Module

## Description
Module for interacting with different blockchain networks and application bridge accounts/contracts.
Implements the bridge logic for the application.

Contains:
- RPC client connection configuration for different blockchain networks;
- Token bridging logic: deposits validation, withdrawals forming and sending;

## Components
- `/chain`: Module for configuring specific blockchain network connection and additional bridging params;
- `/clients`: Module for interacting with the blockchain networks and bridges:
  - `/clients/evm`: Module for interacting with EVM-based networks;
  - `/clients/bitcoin`: Module for interacting with Bitcoin network;
  - `/clients/zano`: Module for interacting with Zano network;

## Supported Networks
Bridge module currently supports:
- EVM-based networks (Ethereum, Binance Smart Chain, etc.);
- Bitcoin;
- Zano.

## Withdrawal Constructor 

### Description
Withdrawal constructor is responsible for:
- forming withdrawal signing data or unsigned transactions based on provided deposit data;
- validating the data to sign that corresponds to the provided deposit data;

Withdrawal constructor is different for each supported network type, as each network has its own unique withdrawal algorithms.

### EVM networks
Signing data construction: according to the provided deposit data, the constructor forms the ERC20/native token withdrawal operation data and hashes it using [`EIP-191 Signed Data Standart`](https://eips.ethereum.org/EIPS/eip-191); the resulting hash is a ready-to-sing data.

Signing data validation: using the provided deposit data and the signing data, the constructor forms the withdrawal operation as in the previous step and compares the resulting hash with the provided one.

### Bitcoin network
Signing data construction: according to the provided deposit data, the `fundrawtransaction` wallet RPC method is called to form a withdrawal transaction.
This will form an unsigned transaction with:
  - selected inputs for funding the withdrawal;
  - outputs for the receiver and change;
  - properly calculated fee.

`fundrawtransaction` method will be called with next parameters:
  - `includeWatching` - `true` to include only watch-only addresses (TSS pubkey) in the transaction;
  - `changeAddress` - the address to send the change to, set to the TSS pubkey hash;
  - `changePosition` - the position of the change output, set to the first index (second position).
  - `feeRate` - the fee rate in BTC/kvB, set to the default value (0.00001000 BTC per kB);

**NOTE: Bitcoin wallet should be configured to track only UTXOs available to be spent by the TSS private key (TSS pubkey watch-only mode).**

As we do not know scriptPubKey of each input (to form a signing data by TSS parties), the constructor have to execute `listunspent` RPC method, filter used UTXOs and get their scriptPubKey.
Then, the constructor forms a signature hash for each input using the `SIGHASH_ALL` flag.
The resulting array of signature hashes is a ready-to-sign data.

Signing data validation: using the provided deposit data and the signing data, the constructor begins transaction validation with next steps:
1. `listunspent` wallet RPC method is called to get the list of all available UTXOs;
2. For each input in the withdrawal transaction, the constructor checks 
   - if the UTXO is present in the list of available UTXOs;
   - if the UTXO is not used twice in the transaction.
   - constructed signature hash is equal to the provided one by the proposer.
3. Check if the first output contains valid receiver PubKey script and withdrawal amount;
4. Check if the second output contains valid change PubKey script (TSS pubkey hash);
5. Ensure that no other outputs are present in the transaction.
6. Check that transaction fees are calculated correctly:
   - calculate the actual fee by subtracting the sum of the outputs from the sum of the inputs;
   - get the expected transaction size by firstly mocking signature scripts with fake signatures;
   - calculate the fee rate by dividing the actual fee by the transaction size;
   - compare the calculated fee rate with the default one: if the tolerance (10% of the default fee rate) is exceeded, the transaction is considered invalid.

### Zano network
Signing data construction: according to the provided deposit data, the [`emit_asset`](https://docs.zano.org/docs/build/rpc-api/wallet-rpc-api/emit_asset/) request is sent to the Zano wallet RPC server, and the resulting `VerifiedTxID` field is a ready-to-sign data.

Signing data validation: using the provided deposit data and provided additional data from the `emit_asset` response by the proposer, the constructor has the ability to decrypt transaction details using [`decrypt_tx_details`](https://docs.zano.org/docs/build/rpc-api/daemon-rpc-api/decrypt_tx_details) method. 
Constructor validates:
  - if the provided `VerifiedTxID` matches the `decrypt_tx_details` response;
  - if the amount of tokens to be minted is correct;
  - if the token receiver address is correct;
  - if no additional outputs are present in the transaction (except the change).
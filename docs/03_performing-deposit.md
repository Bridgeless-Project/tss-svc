# Performing deposit

## EVM networks
To initiate a transfer from an EVM network, the user should execute either `depositERC20` or `depositNative` functions.

`depositERC20` function:
```solidity
function depositERC20(
    address token_, // token address that should be transferred
    uint256 amount_, // amount of tokens to transfer
    string calldata receiver_, // receiver address on the target network
    string calldata network_, // destination network identifier
    bool isWrapped_ // if the token is wrapped or not
)
```

Note:
- before executing the `depositERC20` function, the user should approve the contract to spend the amount of tokens that should be transferred;
- to obtain the information about the available tokens to transfer, their addresses, chain identifiers and more, query the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core) [`bridge`](https://github.com/Bridgeless-Project/bridgeless-core/tree/main/x/bridge) module.

`depositNative` function:
```solidity
function depositNative(
    string calldata receiver_, // receiver address on the target network
    string calldata network_ // destination network identifier
) payable
```

After the transaction execution, the according event will be emitted, either `DepositedERC20` or `DepositedNative`.

To initiate the transfer processing, the user should provide any of the available parties with the deposit operation data:
- transaction hash—the hash of the transaction that contains the deposit operation;
- transaction nonce—the emitted event index, containing the information about the deposit operation and transfer memo;
- source chain id—the identifier of the source chain where the deposit operation was executed.

## Bitcoin

To initiate a transfer from the Bitcoin network, the user should construct a transaction aligning with the next requirements:
- deposit transaction should contain the VOUT-X (x is the index of the output) pointed to the TSS network account address (P2PKH).
The amount of the output will be tracked as the deposit amount and should not be below the dust threshold (1000 sats);
- the transaction should contain the memo with the required information about transfer parameters (destination address, chain id etc.) to be processed by the TSS network.
It should be included as VOUT-(X+1) output using the OP_RETURN script.
As the OP_RETURN script is limited to 80 bytes, the memo should be abbreviated and contain only the required information.
  - For EVM networks, the memo should contain the destination address and the destination network identifier. Example: `0x0000..000-35443`, where `0x0000..000` is the destination address and `35443` is the destination network identifier.
  - For Zano network, the memo should contain the Base58-decoded destination address (as in the default format it exceeds the 80 bytes of memo) and the destination network identifier. Example: `addr..-35443`, where `addr..` is the Base58-decoded destination address and `35443` is the destination network identifier.

After the transaction is broadcast, the user should provide the TSS network with the deposit operation data:
- transaction hash—the hash of the transaction that contains the deposit operation, prepended with the `0x` prefix (if not present);
- transaction nonce—the number of the output X that contains the deposit amount. The transaction memo can then be found by checking the next (VOUT-(X+1)) output;
- source chain id—the identifier of the source chain where the deposit operation was executed.

## Zano

To initiate a transfer from the Zano network, the user should construct a transaction aligning with the next requirements:
- the transaction type should be a [`burn_asset`](https://docs.zano.org/docs/build/rpc-api/wallet-rpc-api/burn_asset) transaction;
- the amount of burned asset and its identifier will be tracked as the deposit amount and token;
- the transaction should contain the memo (located in `service_entries` array) with the required information about transfer parameters (destination address, chain id etc.) to be processed by the TSS network.
It should be the present in the Base64-decoded string format of the following structure:

``` go
type DestinationData struct {
	Address string `json:"dst_add"`
	ChainId string `json:"dst_net_id"`
}
```
- transaction should be pointed to TSS network account address using the `point_tx_to_address` transaction field.
In this case, the burning transaction will be visible and processable in the TSS network.

After the transaction is broadcast, the user should provide the TSS network with the deposit operation data:
- transaction hash—the hash of the transaction that contains the deposit operation, prepended with the `0x` prefix (if not present);
- transaction nonce—the index of `service_entries` array item with transfer destination information;
- source chain id—the identifier of the source chain where the deposit operation was executed.

## TON
To initiate a transfer from the TON network, the user can select one of 3 available methods:
- Native TON deposit;
- Wrapped Jetton deposit;
- Jetton deposit;

### Native TON deposit
The user should construct the follow message and send it directly to the bridge address.
```protobuf
message(0x5386a723) MsgDepositNative { 
    receiver: Slice; // the address on the destination network (127 bits)
    network: Slice; // the name of the destination network (256bits)
    // the amount of tokens is stored on the context 
    // the sender as well
}
```
The contract validates that the deposit has enought value to cover fee and then emits the `NativeDeposited` event.


### Wrapped Jetton deposit
The user should construct the `MsgJettonBurn` message and send it to its own JettonWallet smart contract. All deposit data are stored on the `forwardPayload`
```protobuf
message(0x595f07bc) MsgJettonBurn {
    queryId: Int as uint64;
    amount: Int as coins; // the amount of jettons to burn
    responseDestination: Address?; // user address to receive the execut message

    forwardAddress: Address?; // the Bridge address 
    forwardTonAmount: Int as coins; // the amount of TON to be sent to the Bridge address(to cover fee)
    forwardPayload: Slice as remaining; // the payload to be sent to the Bridge address
}
```

### Jetton deposit
The user should construct the `MsgJettonTransfer` message and send it to the JettonWallet smart contract.  
```protobuf
message(0xf8a7ea5) MsgJettonTransfer {
    queryId: Int as uint64;
    amount: Int as coins;
    destination: Address;
    responseDestination: Address?;
    customPayload: Cell? = null;
    forwardTonAmount: Int as coins;
    forwardPayload: Slice as remaining;
}
```

To initiate the transfer processing, the user should provide any of the available parties with the deposit operation data:
- transaction hash—the hash of the transaction that contains the deposit operation;ß
- instead of tx_nonce user have to send message logical time that initiating the message's position in the event sequence. Learn more about [logical time](https://docs.ton.org/v3/documentation/smart-contracts/message-management/messages-and-transactions/#what-is-a-logical-time);
- source chain id—the identifier of the source chain where the deposit operation was executed.


## Solana

To initiate a transfer from Solana network, the user should invoke one of the following instructions on the bridge program:

### `DepositNative` 
`DepositNative` instruction is used to deposit SOL (lamports) and requires the following accounts and parameters:
``` rust
bridge_id: String,
amount: u64,        // amount of tokens to transfer (18 decimals)
chain_id: String,   // destination network identifier
address: String,    // receiver address on the target network
```

### `DepositSpl`
`DepositSpl` instruction is used to deposit non-wrapped tokens and requires the following accounts and parameters:
``` rust
bridge_id: String,
amount: u64,        // amount of tokens to transfer
chain_id: String,   // destination network identifier
address: String,    // receiver address on the target network

mint: InterfaceAccount<'info, Mint>,
sender: InterfaceAccount<'info, TokenAccount>,
```

- `mint` is a Mint account of the token to be sent. Before depositing tokens of a given mint, 
an auxiliary instruction `InitSplVault` needs to be used by bridge admin with that mint.
- `sender` is a TokenAccount of the given Mint, not necessarily an associated one, belonging to the signer.

### `DepositWrapped`
`DepositWrapped` instruction is used to deposit wrapped (bridge-owned) tokens and requires the following accounts and parameters:
``` rust
bridge_id: String,
mint_nonce: u64,
symbol: String,
amount: u64,        // amount of tokens to transfer (18 decimals)
chain_id: String,   // destination network identifier
address: String,    // receiver address on the target network

sender: InterfaceAccount<'info, TokenAccount>,
```
- `symbol` and `mint_nonce` of a token must be the same as configured on core blockchain,
  (nonce exists to prevent symbol squatting by malicious actors). 
- `sender` is a TokenAccount of the Mint, not necessarily an associated one, belonging to the signer.

`bridge_id` is an arbitrary constant string, set on the TSS service side. 
To be successfully processed, a deposit must contain an identical one.

Some used accounts are derived automatically and are not described here.


After the transaction is broadcast, the user should provide the TSS network with the deposit operation data:

- transaction hash — the hash of the transaction that contains the deposit operation 
(on Solana - the first signature in tx, in Base58 encoding);
- transaction nonce — the index of the target instruction in the tx 
(the first one if called directly);
- source chain id — the identifier of the source chain where the deposit operation was executed.

# Bridging Parameters
To find the required information about the supported tokens and chains, the user should query the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core) [`bridge`](https://github.com/Bridgeless-Project/bridgeless-core/tree/main/x/bridge) module, which contains the information about the available tokens, their addresses, chain identifiers and more.
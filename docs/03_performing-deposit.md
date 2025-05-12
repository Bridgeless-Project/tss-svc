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
- to obtain the information about the available tokens to transfer, their addresses, chain identifiers and more, query the Cosmos [Bridge Core](https://github.com/hyle-team/bridgeless-core) [`bridge`](https://github.com/hyle-team/bridgeless-core/tree/main/x/bridge) module.

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

# Bridging Parameters
To find the required information about the supported tokens and chains, the user should query the Cosmos [Bridge Core](https://github.com/hyle-team/bridgeless-core) [`bridge`](https://github.com/hyle-team/bridgeless-core/tree/main/x/bridge) module, which contains the information about the available tokens, their addresses, chain identifiers and more.
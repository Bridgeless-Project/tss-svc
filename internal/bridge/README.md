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
- For EVM networks:
  - Signing data construction: according to the provided deposit data, the constructor forms the ERC20/native token withdrawal operation data and hashes it using [`EIP-191 Signed Data Standart`](https://eips.ethereum.org/EIPS/eip-191); the resulting hash is a ready-to-sing data.
  - Signing data validation: using the provided deposit data and the signing data, the constructor forms the withdrawal operation as in the previous step and compares the resulting hash with the provided one.
- For Bitcoin network:
  - **TODO:ADD**
- For Zano network:
  - Signing data construction: according to the provided deposit data, the [`emit_asset`](https://docs.zano.org/docs/build/rpc-api/wallet-rpc-api/emit_asset/) request is sent to the Zano wallet RPC server, and the resulting `VerifiedTxID` field is a ready-to-sign data.
  - Signing data validation: using the provided deposit data and provided additional data from the `emit_asset` response by the proposer, the constructor has the ability to decrypt transaction details using [`decrypt_tx_details`](https://docs.zano.org/docs/build/rpc-api/daemon-rpc-api/decrypt_tx_details) method.
  Constructor validates:
    - if the provided `VerifiedTxID` matches the `decrypt_tx_details` response;
    - if the amount of tokens to be minted is correct;
    - if the token receiver address is correct;
    - if no additional outputs are present in the transaction (except the change).
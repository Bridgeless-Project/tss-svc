# /internal/bridge

## Description
Module for interacting with different blockchain networks and application bridge accounts/contracts.
Implements the bridge logic for the application.

Contains:
- RPC client connection configuration for different blockchain networks;
- Token bridging logic: deposits validation, withdrawals forming and sending;

## Components
- `/chain`: Module for configuring specific blockchain network connection and additional bridging params;
- `/clients`: Module for interacting with the blockchain networks and bridges;

## Supported Networks
Bridge module currently supports:
- EVM-based networks (Ethereum, Binance Smart Chain, etc.);
- Bitcoin;
- Zano.
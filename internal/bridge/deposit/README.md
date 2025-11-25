# Deposit fetcher

## Description

Deposit fetcher submodule is designed to fetch deposit data and check deposit existence on core using provided network rpc connection and [Bridge Core connector](../../core/README.md#connector)

## Fetching data

Fetching data is performed through next steps:
- configuring source chain client with chain id from given deposit identifier
- validating deposit identifier data using source client
- fetching deposit data using rpc from client
- receiving source token info
- receiving token pair for deposit token, using source chain id, source token address  and core connector
- receiving destination token info
- forming withdrawal amount using tokens info

In the end of fetching, user receives ready-to-insert deposit, with all necessary data.
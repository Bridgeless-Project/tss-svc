# Bridge Core

## Description
Module for interacting with the Cosmos [Bridge Core](https://github.com/hyle-team/bridgeless-core), especially with its `bridge` module

## Components
To be added

### Connector
Core connector module is designed to query and save bridging data to the Cosmos Bridge Core.

### Catch-Upper
Catch-upper module is designed to catch up with the processed transfers saved on Cosmos Bridge Core.
When the TSS party goes down, to start signing again the pending transfers, the catch-upper should sync the processed transfers from the Cosmos Bridge Core to prevent double-spending and party misfunctioning.

### Subscriber
Subscriber module is designed to listen to the Cosmos Bridge Core events, especially the newly processed transfers.
As the party can not be included to the current session signers set, the subscriber should listen to the processed transfers and notify the party to update its internal state.


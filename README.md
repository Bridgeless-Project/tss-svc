# TSS Service

Threshold signature service provides the processing and signing of the deposited token transfers to another chains based on the threshold signature scheme (TSS).
It works as a decentralized solution connected to the Cosmos [Bridge Core](https://github.com/hyle-team/bridgeless-core) module to process and accumulate the cross-chain transfers.

The TSS network requires a several parties launched by different validators.
In a cooperation with each other, they will process the incoming deposits and withdraw user funds to the destination chain.

Fore more information check the [`TSS Overview`](./docs/01_overview.md).

## Becoming a part of TSS network
Currently, to become a TSS network party, you should follow the next steps:
- be a validator of the Cosmos [Bridge Network](https://github.com/hyle-team/bridgeless-core);
- run the full nodes of the supported networks (e.g. Ethereum, Bitcoin, Zano etc.);
- todo:add

## Examples
For configuration, running and usage examples, see the [`Examples`](./examples) folder.
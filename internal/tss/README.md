# TSS module

## Description
TSS module is responsible for the threshold signature scheme (TSS) that is used for signing the data required to perform the cross-chain transfer.

## Table of Contents
- [Module Components](#components)
- [Keygener submodule](#keygener)
- [Distributor submodule](#distributor)
- [Session](#session)
- [Consensus submodule](#consensus)
- [Signer submodule](#signer)
- [Finalizer submodule](#finalizer)
- [Dependencies](#dependencies)

## Components
To be added

---

## Keygener

### Description
Keygener is a submodule for generating the secret shares for the parties in the TSS network and the private party key.
It is responsible for generating the secret shares for the parties and distributing them to the parties in the network.
Generated private party key is used for signing the withdrawal data in the TSS network by all active parties.
The secret shares are generated using the third-party [`tss-lib`](#dependencies) library.

**Note**:
- The main keygen process is performed only once when the network is initialized;
- The keygen process is performed by all parties in the network (they should be active and with the appropriate service mode);
- The keygen process can be reused in case of resharing the secret shares or adding new parties to the network to generate new party private key.

### Inputs
To start the keygen process, the following inputs are required:
- list of active and ready parties to collaborate with;
- generated party pre-parameters (should be generated before starting the keygen process, see [`pre-params generation`](../../cmd/README.md#generation));

### Outputs
Keygener provides the out channel where the parties should send messages.

After the keygen process is completed, the output for the local party is the secret share that is used with other parties to sign the data with the system private party key.

---

## Distributor

### Description
Distributor is a submodule for validating and distributing the incoming transfer deposits to the parties in the TSS network.

As every party in the TSS network is able to receive users' transfer requests, it should be able to distribute the incoming transfer deposits to other parties in the network.
This is made to:
- accelerate the process of TSS signing by backgrounded deposit validation before starting the signing process;
- prevent the situation when only the small group of parties receives the enormous majority of the transfer deposits. It can lead to the situation when proposer party (see [Consensus](#consensus) module) does not have anything to propose for signing and session is stuck for a while.

Invalid deposits should be rejected and not distributed to the parties in the network.

### Inputs
To start the deposit distribution process, the following inputs are required:
- list of active and ready parties to collaborate with;
- healthy database connection;
- incoming deposit identifiers.

### Outputs
Distributor provides the out channel where incoming deposits should be sent.

---

## Session

### Description
Session is a submodule for managing the TSS session lifecycle.

### Signing session
A TSS signing session is a set of operations that are performed by the parties in the network to process the withdrawal request.
The main goal of the session is to define the current signers set, the data to be signed, sign the data, and finalize the transfer process.

Session consists of the following steps:
1. `Acceptance` - reaching an agreement between the parties in the TSS network on the data to be signed next. Uses the [Consensus](#consensus) submodule;
2. `Signing` - signing the provided data by communicating with other parties in the TSS network. Uses the [Signer](#signer) submodule;
3. `Finalization` - finalizing the signing process by saving data/executing other finalization steps. Uses the [Finalizer](#finalizer) submodule.

There are as many active sessions as the total number of supported chains in the system.
Each session is responsible for processing the withdrawal requests on the specific chain.
This is done to:
- prevent the mixing of the withdrawal requests from the same chains (f.e trying to sign the same data twice, use the same UTXO in different transactions etc);
- speed up the signing process by parallelizing the non-conflicting withdrawal requests processing.

Each session has its own lifecycle and identifier that changes with each new session.
New session in this context is an old finished session with new (incremented) session identifier that is ready to process new withdrawal requests (for the same chain as previous session) and waits for its start.

### Keygen session
Keygen session is a special session that is used to generate the secret shares for the parties in the TSS network.
It is performed only once when the network is initialized and the parties are ready to start the TSS signing process.


### Session boundaries
To control the session duration, the session should be bounded by the time limits.
Those limits are different for each step of the session (acceptance, signing, finalization) and session chain type.
Also, each active session changes to the new one once in a constant time interval.

Here is the list of the signing session time bounds:
- EVM session:
  - acceptance: 10 seconds;
  - signing: 10 seconds;
  - finalization step: 10 seconds;
  - new session period: 30 seconds.
- Zano session:
  - acceptance: 10 seconds;
  - signing: 10 seconds;
  - finalization step: 10 seconds;
  - new session period: 30 seconds.
- Bitcoin session:
  - acceptance: 10 seconds;
  - signing: 10 seconds * number N of UTXOs to be signed in the transaction;
  - finalization step: 10 seconds;
  - new session period: 60 seconds.

In case of the session step timeout, the session should be finished and the new session should be initialized and wait for its start.

Keygen session deadline is 1 minute.

### Session manager
Session manager is responsible for managing the set of sessions.
It is responsible for:
- providing the specific session with other parties session messages;
- providing the requestor with the specific session information;

### Bitcoin signing session
Bitcoin signing session is a special session that is used to process the Bitcoin withdrawals.
It is different from the EVM and Zano sessions as it requires multiple UTXOs to be signed independently.
Thus, the session includes N signing rounds, where N is the number of UTXOs to be signed in the transaction.
Respectively, the signing step time bounds are multiplied by N to provide enough time for the parties to sign all UTXOs.

In order to prevent the growing number of UTXOs to be signed in the session, the consolidation process session is run periodically.
Consolidation session is a special session that is used to group the larger number of UTXOs in one transaction with a few outputs to reduce the number of UTXOs to be signed in the future sessions.
It is triggered automatically by reaching the threshold number of UTXOs and the next pending withdrawal request will be processed right after the consolidation process is finished.

### Catchup
For the initial sessions start, the parties are required to have the same session start time and initial session identifier.

In case when some party lost the connection and misses current session data, it should request the session information from other parties.
Session information can include:
- current session identifier;
- session start time;
- session deadline;

Using this information, the party can calculate the current session identifier and session time bounds and catch up with the other parties by waiting for the current session deadline.

---

## Consensus

### Description
Consensus is a submodule for reaching an agreement between the parties in the TSS network on the data to be signed next.
It can be used for arbitrary data that should be signed by the parties in the network, f.e. withdrawals, resharing transactions, etc.

### Mechanism
Consensus mechanism is based on the proposer selection and the data sharing between the parties in the network.
There is two possible roles for the party in the consensus process:
- `proposer` - the party that selects the data to be signed and shares it with all parties in the network;
- `acceptor` - the party that validates and accepts the data shared by the proposer;

Only one proposer is selected for the current signing session, while all other parties are acceptors.
At the end of the consensus process, the proposer selects the signers set that will sign the data.
Proposer deterministically selects the signers set from acceptors set that ACKed the signing request.
Proposer is included in the signers set as well.
Signers count is always equal to the signing threshold value.

The consensus process is performed by the following steps:
1. All parties in the network should deterministically choose the proposer for the current signing session.
Proposer is selected using the deterministic function `f(session_id)` using the [`ChaCha8`](#dependencies) pseudo-random number generator.
2. Proposer forms the signing proposal (f.e. withdrawal on the specific chain) based on the provided mechanism.
It shares the constructed data and metadata with all parties in the network.
If there is no data to be signed, proposer broadcasts the no-signing-data message and process is finished.
3. Parties that received proposer request (acceptors) should validate the proposal request and reply with acknowledgement status:
     - ACK if everything is fine;
     - NACK if something isn't valid (already signed proposal, non-existent deposit, etc.).
4. While acceptors ACKing or NACKing proposer request, the proposer collects all ACKed responses.
It should check that number of ACKs N is equal or bigger than provided threshold value T.
   - if true, proposer deterministically selects the T signers from the N acceptors that ACKed the signing request (including proposer itself).
   They are notified about being included to the signer set. 
   - if false, the process finishes.
5. Notified acceptors receive the current signing set and can additionally validate that all parties forming the signers set are valid and active.
Acceptors that are not included in the current signers set can wait till consensus deadline and understand that they are not the part of the current signers set.
Optionally, they can be notified by proposer that they won't take part in current signing process.

### Inputs
To start the consensus process, the following inputs are required:
- list of active and ready parties to collaborate with;
- implemented consensus forming/validation mechanism functions to:
  - form the signing proposal;
  - validate the signing proposal;
  - parse the signing proposal;

### Outputs
After the consensus process is completed, the output is the data to be signed and the list of parties that will sign the data.
If the party is not included in the signers list, the signing data will be empty, and it should wait for the next session.
---

## Signer

### Description
Signer is a submodule for signing the provided data by communicating with other parties in the TSS network.
P2P communication is provided by the [P2P module](../p2p/README.md), while the TSS signing is provided by the third-party [`tss-lib`](#dependencies) library.

**Note:**
- No signing data validation is performed in this module;
- All parties should start the signing process at the same time;
- All parties should sing exactly the same data;
- There are enough parties to reach the threshold.

It is assumed that the data is validated before being passed to the signer and all parties agreed on the data to be signed.

### Inputs
To start the signing process, the following inputs are required:
- data to be signed;
- list of parties to collaborate with (or broadcaster to send the data to all parties);
- signing threshold;
- local party secret share.


### Outputs
After the signing process is completed, the output is the signature of the data and the error if any (timeout, not enough parties, signing error, etc.).

---

## Finalizer

### Description
Finalizer is a submodule for finalizing the signing process.
It is responsible for saving the signed transfers to the [Bridge Core](../core/README.md).
Additionally, it can be used to broadcast the signed transfers to the network or do other finalization steps (different for each chain).

### Finalization process

#### EVM networks
For EVM networks, the finalization process is performed only by saving the signed withdrawal data to the [Bridge Core](../core/README.md).
Then it can be used by anyone to construct and broadcast the withdrawal transaction to the destination network.

**Note:** TSS network does not broadcast the signed EVM transactions to the network, user should do it manually and pay the gas fee.

#### Bitcoin network
For the Bitcoin network, the finalization process, in addition to saving the signed withdrawal data to the [Bridge Core](../core/README.md), also broadcasts the signed transaction to the Bitcoin network.

#### Zano network
For the Zano network, the finalization process, in addition to saving the signed withdrawal data to the [Bridge Core](../core/README.md), also broadcasts the signed transaction to the Zano network.


**Note:** currently, the finalization process should be performed by the session proposer, see [Consensus](#consensus) for more details;

### Inputs
To start the finalization process, the following inputs are required:
- signed transfer data;
- Bridge Core connection;
- optional data for finalization (different for each chain).

### Outputs
Finalizer does not provide any outputs, except for the finalization error if any.

---

## Dependencies
- [tss-lib](https://github.com/bnb-chain/tss-lib) - a library for threshold signature scheme (TSS) that is used for signing the data.
- [ChaCha8](https://pkg.go.dev/math/rand/v2) - a pseudo-random number generator using ChaCha8 algorithm from the new official Go `math/rand/v2` package.

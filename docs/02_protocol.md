# TSS Protocol

## Key Generation
Before starting the TSS signing process, parties should generate the general system private key.
It will be used to sign the transactions or data required to perform the cross-chain transfer.
Actually, the private key is not generated directly.

By communicating with each other, parties generate the secret shares of the private key that, if combined (and number of shares is bigger than threshold), will be able to sign the provided data just like using the private key.
As the result, each party will have its own secret share of the private key, which they should keep secure and in secret.
The secret shares are generated using the third-party [tss-lib](https://github.com/bnb-chain/tss-lib) library.

**Note**:
- The keygen process is performed by all parties in the network (they should be active and set with the appropriate service mode);
- The keygen process can be reused in case of resharing the secret shares or adding new parties to the network to generate the new party private key.

After the keygen process is completed, the output for the local party is the key secret share that is used with other parties to sign the data with the system private party key.

## TSS Signing

### Signing sessions
TSS signing process is performed as a series of signing sessions, which are responsible for signing the data required to perform the specific transfer.
A TSS signing session is a set of operations that are performed by the parties in the network to process the withdrawal request.
The main goal of the session is to define the current signers set, the data to be signed, sign the data, and finalize the transfer process.

There are as many active sessions as the total number of supported chains in the system.
Each session is responsible for processing the withdrawal requests on the specific chain.
This is done to:
- prevent the mixing of the withdrawal requests from the same chains (f.e trying to sign the same data twice, use the same UTXO in different transactions etc);
- speed up the signing process by parallelizing the non-conflicting withdrawal requests processing.

Each session has its own lifecycle and identifier that changes with each new session.
New session in this context is an old finished session with new (incremented) session identifier that is ready to process new withdrawal requests (for the same chain as previous session) and waits for its start.

Each session has a leader party that is responsible for selecting the data to be signed,
the signers set, and the finalization process.
The leader party for the specific session is selected deterministically,
meaning each party knows who will be the leader for the next session.

Signing session process can be divided into three main stages:
1. `Acceptance` - reaching an agreement between the parties in the TSS network on the data to be signed next;
2. `Signing` - signing the provided data by communicating with other parties in the TSS network;
3. `Finalization` - finalizing the signing process by saving data/executing other finalization steps.


#### Acceptance
To start signing operations, parties should reach a consensus on arbitrary data that should be signed, f.e. withdrawals, resharing transactions, etc.
There are two possible roles for the single TSS party in the consensus process:
- `proposer` - the party that selects the data to be signed and shares it with all parties in the network;
- `acceptor` - the party that validates and accepts the data shared by the proposer;

Only one proposer is selected for the current session (session leader), while all other parties are acceptors.
At the end of the consensus process, the proposer selects the signers set that will sign the data.
Proposer deterministically selects the signers set from acceptors set that ACKed the signing request.
Proposer is included in the signers set as well.
Signers count is always equal to the signing threshold value (plus one).

The following steps perform the consensus process:
1. Proposer forms the signing proposal (f.e. withdrawal on the specific chain) based on the provided mechanism.
   It shares the constructed data and metadata with all parties in the network.
   If there is no data to be signed, proposer broadcasts the message with empty data and the consensus process is finished.
2. Parties that received proposer request (acceptors) should validate the proposal request and reply with acknowledgement status:
    - ACK if everything is fine;
    - NACK if something isn't valid (already signed proposal, non-existent deposit, etc.).
3. While acceptors ACKing or NACKing proposer request, proposer collects all ACKed responses.
   It should check that the number of ACKs N is equal or bigger than the provided threshold value T:
    - if true, proposer deterministically selects the T signers from the N acceptors that ACKed the signing request (including itself).
    - if false, the consensus process finishes.
4. Proposer notifies all parties in the network about the selected signers set and the data to be signed.
   Each party can additionally validate the selected signers set.

After the consensus process is completed, the output is the data to be signed and the list of parties that will sign the data.
If the party is not included in the signers list, the list will be empty, and it should wait till the next session will be started.

#### Signing
After the data is accepted, the signing process, based on [tss-lib](https://github.com/bnb-chain/tss-lib) ECDSA signing rounds, is started.

**Note:**
- No signing data validation is performed at this step;
- All parties should start the signing process at the same time;
- All parties should sing exactly the same data;
- There are enough parties to reach the threshold.

After the signing process is completed, the output is the signature of the data and the error if any (timeout, not enough parties, signing error, etc.).
Then, the session leader distributes the obtained signature to all parties in the network.
Each party can ensure the signature is valid and matches the previously obtained data to be signed.

#### Finalization
After the data is signed by the required number of parties, the final signature is produced and sent to the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core).
Additionally, it can be used to broadcast the signed transfers to the network or do other finalization steps (different for each chain).

##### EVM networks
For EVM networks, the finalization process is performed only by saving the signed withdrawal data to the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core).
Then it can be used by anyone to construct and broadcast the withdrawal transaction to the destination network.

**Note:** TSS network does not broadcast the signed EVM transactions to the network, user should do it manually and pay the gas fee.

##### Bitcoin network
For the Bitcoin network, the finalization process, in addition to saving the signed withdrawal data to the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core)., also broadcasts the signed transaction to the Bitcoin network.

##### Zano network
For the Zano network, the finalization process, in addition to saving the signed withdrawal data to the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core)., also broadcasts the signed transaction to the Zano network.

#### TON network
For TON, the finalization process is performed only by saving the signed withdrawal data to the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core).
Then it can be used by anyone to construct and broadcast the withdrawal transaction to the destination network.

##### Solana network
For the Solana network, the finalization process is performed by saving the signed withdrawal data to the Cosmos [Bridge Core](https://github.com/Bridgeless-Project/bridgeless-core). 
Then it can be used by anyone to construct and broadcast the withdrawal transaction.
Note that it is not a fully assembled transaction that is being signed, but a structure with the withdrawal parameters
(e.g. amount, receiver).

**Note:** currently, the finalization process should be performed by the session proposer.

---

After the session signing process is finished, parties should start the new session to sign the next transfer.

### Bitcoin signing session
Bitcoin signing session is a special session used to process the Bitcoin withdrawals.
It is different from the EVM and Zano sessions as it requires multiple UTXOs to be signed independently.
Thus, the session includes N signing rounds, where N is the number of UTXOs to be signed in the transaction.
Respectively, the signing step time bounds are multiplied by N to provide enough time for the parties to sign all UTXOs.

To prevent the growing number of UTXOs from being signed in the session, the consolidation process session is run periodically.
Consolidation session is a special session that is used to group the larger number of UTXOs in one transaction with a few outputs to reduce the number of UTXOs to be signed in the future sessions.
It is triggered automatically by reaching the threshold number of UTXOs and the next pending withdrawal request will be processed right after the consolidation process is finished.


### Session catchup
For the initial sessions start, the parties are required to have the same session start time and initial session identifier.

In case when some party lost the connection, it requests the session information from other parties for each existing session.
Session information can include:
- current session identifier;
- next session start time.

Using this information, the party calculates the next session identifier, current session time bounds,
and catches up with the other parties by waiting for the current session deadline.

## Synchronization
To prevent system failures, reach the consensus, and ensure the correct signing process, parties should be synchronized with each other.
The synchronization process is based on using timestamps for each session duration and its steps.
The time bounds are strongly defined for each session stage and type, so the result session duration is also constant (although there can be some exceptions).
To control the session duration, the session should be bounded by the time limits.

Here is the list of the signing session time bounds:
- EVM session:
    - acceptance: 20 seconds;
    - signing: 20 seconds;
    - finalization step: 15 seconds;
    - new session period: 60 seconds.
- Zano session:
    - acceptance: 20 seconds;
    - signing: 20 seconds;
    - finalization step: 15 seconds;
    - new session period: 60 seconds.
- Bitcoin session:
    - acceptance: 20 seconds;
    - signing: 20 seconds * number N of UTXOs to be signed in the transaction;
    - signing rounds delay (if more than 1 singing round needed): 500 ms;
    - finalization step: 15 seconds;
    - new session period: calculated based on the number of UTXOs to be signed in the transaction.
- TON session:
    - acceptance: 20 seconds;
    - signing: 20 seconds;
    - finalization step: 15 seconds;
    - new session period: 60 seconds.
  
In case of the session step timeout, the session should be finished,
and the new session should be initialized and wait for its start.

Keygen session deadline is 1 minute.
Resharing session deadline calculated based on the amount of data needed to be processed.

--- 

## Key Resharing
To ensure the system scalability and security, parties can join or leave the TSS network.
It means that the secret shares of the general system private key should be redistributed among the old/new parties.
The change in the number of parties can cause the change of the threshold value for the number of signers required to sign the data.
In that case, the key resharing process cannot be executed by the means of resharing process from the [tss-lib](https://github.com/bnb-chain/tss-lib) library.

Moreover, the change of private key shares causes the additional processes of fund migration and ecosystem reconfiguration.
Thus, the key resharing process is not performed automatically and should be handled manually by the system administrators in cooperation with the network parties.

---

## Security
Protocol security satisfies the [TSS library recommendations](https://github.com/bnb-chain/tss-lib?tab=readme-ov-file#how-to-use-this-securely).

### Secret shares
To ensure the system security, parties should keep their secret shares of the private key in secret and secure.
The secret shares should not be shared with anyone, even with the other parties. For this purpose, they are securely stored in the [Vault](../internal/secrets/README.md) and accessed only when required.

### Transport
Communication (messages transport) between parties is built using mTLS over gRPC, which ensures the secure data exchange between parties.
It ensures that both parties are authenticated and the data is encrypted during the transmission.

Additionally, within the transport, each message is wrapped with a session ID that is unique to a single run of the keygen, signing or re-sharing session.
This session ID is agreed upon out-of-band and known only by the participating parties before the session begins.
For a series of signing sessions, the session ID is incremented by one for each next session.

### Broadcast
There is a mechanism in transport that allows "reliable broadcasts", meaning parties can broadcast a message to other parties
such that it's guaranteed each one receives the same message. Reliable broadcasts are implemented based on the 
[Dolev-Strong algorithm](https://www.cs.huji.ac.il/~dolev/pubs/authenticated.pdf).

As the message validation is not the part of the algorithm, the session ID is used to secure the signing message.
It prevents the malicious parties from replacing the original proposer message and signature with the
valid one from the previous sessions.

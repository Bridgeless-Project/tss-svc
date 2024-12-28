# TSS Protocol

## Key Generation
Before starting the TSS signing process, parties should generate the general system private key.
It will be used to sign the transactions or data required to perform the cross-chain transfer.
Actually, the private key is not generated directly.
By communicating with each other, parties generate the secret shares of the private key that, if combined (and number of shares is bigger than threshold), will be able to sign the provided data just like using the private key.
As the result, each party will have its own secret share of the private key, which they should keep secure and in secret.

Check the [keygener](../internal/tss/README.md#keygener) documentation for more details on key generation process.

## TSS Signing
TSS signing process is performed as a series of signing sessions, which are responsible for signing the data required to perform the specific transfer.
See the [session](../internal/tss/README.md#session) documentation for more session details.

Signing session process consists of the following steps:

### Acceptance
To start signing the data, parties should firstly define and accept the transfer to be signed.
[Consenus](../internal/tss/README.md#consensus) module is responsible for choosing the data to be signed and the list of current session signers.
Check it for more details on how the data is chosen and how the signers are defined.

### Signing
After the data is accepted, parties should start the process of signing it and communicate with each other to produce the final transfer signature.
This process is handled by the [Signer](../internal/tss/README.md#signer) module, check it for more details.

### Finalization
After the data is signed by the required number of parties, the final signature is produced and sent to the Cosmos [Bridge Core](https://github.com/hyle-team/bridgeless-core).
Additionally, the withdrawal transaction can be broadcast to the network (varies depending on the network).
The finalization process is described in the [Finalizer](../internal/tss/README.md#finalizer) module.

---

After the session signing process is finished, parties should start the new session to sign the next transfer. 

## Synchronization
To prevent system failures, reach the consensus, and ensure the correct signing process, parties should be synchronized with each other.
The synchronization process is based on using timestamps for each session duration and its steps.
The time bounds are strongly defined for each session stage and type, so the result session duration is also constant (although there can be some exceptions).
See [Session boundaries](../internal/tss/README.md#session-boundaries) for more details on time bounds.

---

## Key Resharing
To ensure the system scalability and security, parties can join or leave the TSS network.
It means that the secret shares of the general system private key should be redistributed among the old/new parties.
The change in a number of parties can cause the change of the threshold value for number of signers required to sign the data.
In that case, the key resharing process cannot be executed by the means of [tss-lib](https://github.com/bnb-chain/tss-lib) library.
Moreover, the change of private key shares causes the additional processes of funds migration and ecosystem reconfiguration.
Thus, the key resharing process is not performed automatically and should be handled manually by the system administrators in a cooperation with the network parties.
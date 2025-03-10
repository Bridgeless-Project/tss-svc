# Key resharing

## Description
Key resharing is the process that is executed when the number of parties in the TSS network is changed.
It involves the whole ecosystem reconfiguration and funds migration to the new accounts.
There are several steps that should be executed one by one to ensure the correct system operation.
TODO: DESCRIBE ALL STEPS

## Steps

### 1. New key generation

### 2. EVM networks reconfiguration

### 3. Bitcoin network reconfiguration

#### 3.x. Funds migration to the new TSS bitcoin account
Once the Bitcoin address was derived from the new general system public key, the funds should be migrated to it.
Each TSS party should be configured with the new funds receiving address and be ready to start the service in Bitcoin resharing mode.
Additionally, parties should agree on the max number of inputs to include for a single migration transaction and a tx commission rate before the resharing process wil be started.

If the total amount of UTXOs is greater than the maximum number of inputs, the Bitcoin resharing mode can and should be started several times until all the funds are migrated.

### 4. Zano network reconfiguration

### 5. Bridge module settings reconfiguration

### 6. Local TSS reconfiguration

### 7. TSS Vault secrets reconfiguration 

### 8. System restart
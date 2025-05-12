# Running The Service

## Prerequisites
- The key generation process was successfully completed and the shares were generated and stored in the Vault;
- All blockchain nodes are properly configured and started;
- All required EVM contracts, tokens and assets are deployed and available;
- Database is properly configured, started, and the connection string is set in the configuration file;

## First run system pre-setup steps [System administrator only]

### 1. Fill the Bridge Core module settings
The Bridge Core module settings, parameters, and data should be set up.
This includes:
- parameters: 
  - `tss_threshold` - the threshold value for the TSS network;
  - `parties` - list of parties that are running the TSS service;
- state:
  - `chains` - list of chains that are supported by the TSS network;
  - `tokens` - list of tokens that are supported by the TSS network;

To obtain addresses of the TSS service (bridge address) for each chain, there is a list of CLI
commands that can be executed:

- Retrieving the TSS public key points:
```bash
tss-svc helpers vault get pubkey -c <path-to-config-file>
```
- Parsing the Ethereum address from the public key points:
```bash
tss-svc helpers parse address-eth <x-cord> <y-cord>
```
- Parsing the Bitcoin address (P2PKH) rom the public key points:
```bash
tss-svc helpers parse address-btc <x-cord> <y-cord> --network testnet|mainnet 
```

### 2. Fill the EVM contracts
The TSS service address should be set in the EVM contracts as the signer address.

### 3. Configure ZANO tokens
When the TSS service public key is generated, the Zano tokens should be configured with the new `owner_eth_pub_key` field
to be able to execute the ADO operations using the TSS service signatures.

## First run system pre-setup steps [All parties]

### 1. Configure the Bitcoin wallet [If used]
The Bitcoin wallet should be configured to track the outputs of the TSS service address.
The `importdescriptors` wallet command should be executed to import the TSS service address descriptor for P2PKH transactions.

When importing new address descriptor, remember to use the compressed public key format, f.e. `pkh(027356..00)#[hash]`
To get the compressed public key of the TSS, the next CLI commands can be executed:
- Retrieving the TSS public key points:
```bash
tss-svc helpers vault get pubkey -c <path-to-config-file>
```
- Parsing the public key info:
```bash
tss-svc helpers parse pubkey <x-cord> <y-cord>
```

### 2. Complete the configuration file
The configuration file should be fully completed before starting the TSS service.
This includes complete chains configuration (bridge addresses, confirmations, network, RPC endpoints)

Also, remember to fill up the `tss` configuration section with the start time, session id and threshold values.
Make sure that the `tss` section is the same for all parties.

### 3. Run the database migration
Run the database migration to create the required tables and indexes:
```bash
tss-svc service migrate up -c <path-to-config-file>
```

## Running the service in signing mode
To run the TSS service in signing mode, simply execute the following command:
```bash
tss-svc service run signing -c <path-to-config-file> -o vault
```

After the service is started, the signing sessions begin to process incoming deposits once the session start time is reached.

## Re-connecting to the running parties
In case when some error occurs and the local party was disconnected from the running parties,
simply re-run the service in signing mode with the `--sync` flag:
```bash
tss-svc service run signing -c <path-to-config-file> --sync
```

It will allow synchronizing the signing sessions` data with the running parties and continue processing deposits.
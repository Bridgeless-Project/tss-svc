# /internal/secrets

## Description
Module for managing application confidential and crucial secrets.

Currently, supports only [HashiCorp Vault](https://www.vaultproject.io/) as a secret store.

## Configuration
To connect to the Vault, the following environment variables should be set:
- `VAULT_PATH` - the path to the Vault
- `VAULT_TOKEN` - the token to access the Vault
- `MOUNT_PATH` - the mount path where the application secrets are stored. Note: use the `kv v2` secrets engine 

Next secrets should be set in the Vault key-value storage under the `MOUNT_PATH` for proper service configuration:
- for keygen mode:
  - `keygen_preparams` - TSS pre-parameters for the threshold signature key generation
  - `cosmos_account` - Cosmos SDK account private key (in hex format).
- for signing mode:
  - all the secrets from the keygen mode
  - `tss_share` - TSS key share for the local party threshold signature signing

## Examples
TODO: Add examples
# /internal/secrets

## Description
Module for managing application confidential and crucial secrets.

Currently, supports only [HashiCorp Vault](https://www.vaultproject.io/) as a secret store.

## Configuration
To connect to the Vault, the following environment variables should be set:
- `VAULT_PATH` - the path to the Vault
- `VAULT_TOKEN` - the token to access the Vault
- `MOUNT_PATH` - the mount path where the application secrets are stored


Next secrets should be set in the Vault key-value storage under the `MOUNT_PATH` for proper service configuration:
- `preparams` - TSS pre-parameters for the threshold signature key generation
- TODO: Add more secrets

## Examples
TODO: Add examples
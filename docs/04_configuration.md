# Service configuration
To provide the service with the required settings, you need to:
- create the configuration file;
- run the Vault server with configured application secrets;

## Configuration file

The configuration file is based on the YAML format and should be provided to the service during the launch or commands execution.
It stores the service settings, network settings, and other required parameters.


## Vault configuration

[HashiCorp Vault](https://www.vaultproject.io/) is used to store the most sensitive data like keys, private TSS key shares etc.


### Environment variables
To configure the Vault credentials, the following environment variables should be set:
```bash
VAULT_PATH={path} -- the path to the Vault
VAULT_TOKEN={token} -- the token to access the Vault
MOUNT_PATH={mount_path} -- the mount path where the application secrets are stored
```

Example configuration:
```bash
export VAULT_PATH=http://localhost:8200
export VAULT_TOKEN=root
export MOUNT_PATH=secret
```

# /cmd

## Description
Contains the command-line interface (CLI) for the project

## Components
- `root.go`: Contains the main entry point for the CLI
- `/service`: Contains the commands for running the service in different modes
- `/helpers`: Contains the helper functions useful for application configuration and setup

## Commands

Most of the commands require the `--config` flag to be set to the path of the configuration file
By default, the config file path is set to `config.yaml`. See [Configuration](../docs/04_configuration.md) for more details

---

### Service Commands

#### Database Migrations
- `tss-svc service migrate up`: Migrates the database schema to the latest version
- `tss-svc service migrate down`: Rolls back the database schema to the previous version

#### Run server
- `tss-svc service run keygen`: Runs the TSS service in the keygen mode
- `tss-svc service run signing`: Runs the TSS service in the sign mode

#### Additional commands
- `tss-svc service sign [msg]`: Signs a given message using the TSS service

---

### Helper Commands

#### Generation
- `tss-svc helpers generate preparams`: Generates a new set of pre-parameters for the TSS service.
    By default, newly generated pre-parameters are printed to the standard output.
    Use the `--output` (`-o`) flag with parameter to change the output: 
    - `console`: default output;
    - `file`: write the output to a JSON file, use the `--path` flag to specify the file path, default is `preparams.json`;
    - `vault`: write the output to a HashiCorp Vault (requires a running Vault server and configured environment variables, see [Configuration](../docs/04_configuration.md#environment-variables) for more details).

- `tss-svc helpers generate cosmos-account`: Generates a new Cosmos SDK private key and according account address.
    To change the Bech32 Prefix, use the `--prefix` flag with the desired prefix.
    By default, the generated private key and account address are printed to the standard output.
    Use the `--output` (`-o`) flag with parameter to change the output: 
    - `console`: default output;
    - `file`: write the output to a JSON file, use the `--path` flag to specify the file path, default is `cosmos-account.json`;
    - `vault`: write the output to a HashiCorp Vault (requires a running Vault server and configured environment variables, see [Configuration](../docs/04_configuration.md#environment-variables) for more details).

- `tss-svc helpers generate transaction`: Generates a new transaction based on the given data. 
    It is used for resharing purposes. Should be investigated further. 


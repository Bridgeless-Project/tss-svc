# /cmd

## Description
Contains the command-line interface (CLI) for the project. 
Command-line interface for project is designed to help user to prepare and then run the service.

## Components
- `root.go`: Contains the main entry point for the CLI
- `/service`: Contains the commands for running the service in different modes
- `/helpers`: Contains the helper functions useful for application configuration and setup

## Commands
Some of the commands require the mandatory or optional flags to be passed. See the [Flags](#flags) section for more details about specific flag definition and usage.

### Database Migrations
At the start of server user has to migrate up his db to have possibility to process deposits in right way.
Commands:
- `tss-svc service migrate up`: Migrates the database schema to the latest version
- `tss-svc service migrate down`: Rolls back the database schema to the previous version

Required flags:
- `--config` (can be omitted if the default config file path is used)

### Run server
Service can be run into two modes: keygen and signing.
- Signing mode offers user to take part in signing sessions and proceed incoming deposits.
- Keygen mode is designed to generate user`s shares used in signing process.

Commands:
- `tss-svc service run keygen`: Runs the TSS service in the keygen mode
- `tss-svc service run signing`: Runs the TSS service in the sign mode

Required flags:
- `--config` (can be omitted if the default config file path is used)

Optional flags:
- `--output`

### Sign single message
Commands:
- `tss-svc service sign [msg]`: Signs a given message using the TSS service

Required flags:
- `--config` (can be omitted if the default config file path is used)

Optional flags:
- `--output`

### Generation
- `tss-svc helpers generate preparams`: Generates a new set of pre-parameters for the TSS service.
Optional flags:
  - `--output`
  - `--config`

- `tss-svc helpers generate cosmos-account`: Generates a new Cosmos SDK private key and according account address.
Optional flags:
  - `--output`
  - `--config`

- `tss-svc helpers generate transaction`: Generates a new transaction based on the given data. 
    It is used for resharing purposes. Should be investigated further. 

### Parsing
Commands:
- `tss-svc helpers parse address-btc [x-cord] [y-cord]`: Parses btc address from given point 
- `tss-svc helpers parse address-eth [x-cord] [y-cord]`: Parses eth address from given point
- `tss-svc helpers parse pubkey [x-cord] [y-cord]`: Parses public key from given point

Optional flags:
  - `--network` (Network type (mainnet/testnet), mainnet is used by default)

## Flags
- `--config` (`-c`): Specifies the path to the configuration file. By default, the config file path is set to `config.yaml`. See [Configuration](../docs/04_configuration.md) for more details
- `--output` (`-o`): Specifies the data output type for the command.
  Use the flag with parameter to change the desired output:
    - `console`: stdout, default output;
    - `file`: write the output to a JSON file, use the `--path` flag to specify the file path, default is `cosmos-account.json`;
    - `vault`: write the output to a HashiCorp Vault (requires a running Vault server and configured environment variables. Used alongside with `--config` flag. See [Configuration](../docs/04_configuration.md#environment-variables) for more details).
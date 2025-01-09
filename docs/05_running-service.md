# Running The Service

Service can be run in two main modes: keygen and signing.
Also, the service can execute additional commands like database migrations, message signing, etc.

Check the available commands and flags in the [CLI documentation](../cmd/README.md).

## Keygen mode

Before starting the service in the keygen mode, firstly:
- set up the secrets store. See [Configuring Vault](../docs/04_configuration.md#vault-configuration) for more details.
- make sure the configuration file is set up correctly. See [Configuration file](../docs/04_configuration.md#configuration-file) for more details.
- make sure the keygen session `start_time` and `session_id` are the same for all parties.

To run the service in the keygen mode, execute the following command:

```bash
tss-svc service run keygen -c /path/to/config.yaml -o console|file|vault
```

For example, to run the service in the keygen mode with the `./configs/config.yaml` and output the result (local party private share) to the Vault, run the following command:

```bash
tss-svc service run keygen -c ./configs/config.yaml -o vault
```

## Signing mode

# Service configuration
To provide the service with the required settings, you need to:
- create and fill the configuration file;
- run the Vault server with configured application secrets.

## Configuration file

The configuration file is based on the YAML format and should be provided to the service during the launch or commands execution.
It stores the service settings, network settings, and other required parameters.

Example configuration file:
```yaml
# Logger configuration
log:
  # Log level: debug, info, warn, error, fatal
  level: debug
  # Disable Sentry integration
  disable_sentry: true

# Database configuration
db:
  # Database URL connection string
  url: postgres://tss:tss@tss-1-db:5432/tss?sslmode=disable

# Listeners configuration
listeners:
  # address and port for P2P communication between TSS parties (gRPC)
  p2p_grpc_addr: :8090
  # HTTP gateway address and port to access the API endpoints
  api_http_addr: :8080
  # gRPC address and port to access the API endpoints
  api_grpc_addr: :8085

# TSS parties configuration
parties:
  list:
    # first party configuration
      # Bridge Core address identifier of the active TSS peer
    - core_address: bridge123ex5u9qqmlyzzff278ncsn7rwh65ks0urjyzn
      # gRPC address to connect to
      connection: conn
      # party's TLS certificate to verify the connection
      tls_certificate_path: party1.crt
    # next party configuration
    - core_address: ...

# supported chains configuration
chains:
  list:
    # EVM chains configuration
      # Chain ID, must match the Bridge Core chain ID
    - id: "evm1"
      type: "evm"
      # Node RPC endpoint
      rpc: "your_rpc_endpoint_here"
      # Bridge contract address
      bridge_addresses: "test_address"
      ## Number of confirmations required for the withdrawal to be considered final
      confirmations: 1
    # Zano chain configuration
    - id: "zano1"
      type: zano
      confirmations: 1
      # bridge asset receiver(s)
      bridge_addresses:
        - "ZxDphM9gFU..."
      rpc:
          # Zano node RPC endpoint
          daemon: "your_rpc_endpoint_here"
          # Zano wallet RPC endpoint
          wallet: "your_rpc_endpoint_here"
    # Bitcoin chain configuration
    - id: "btc1"
      type: bitcoin
      bridge_addresses:
        - "tb1pugjwudq39gxnpwwm8xelhaulg3m5arzrw69rwy3rz5trptas63ysga329g"
        - "tb1q5pt47kfu77fyl5szk33n5wv2ttf75ka20aqv9f"
      confirmations: 1
      # Bitcoin network: mainnet or testnet
      network: testnet
      rpc:
        # Bitcoin wallet RPC endpoint
        wallet:
          host: "your_rpc_endpoint_here"
          user: "bitcoin"
          pass: "bitcoin"
        # Bitcoin node RPC endpoint
        node:
          host: "your_rpc_endpoint_here"
          user: "bitcoin"
          pass: "bitcoin"


# TSS configuration
tss:
  # session start time (should be in the future)
  start_time: "2025-01-21 15:28:00"
  # session Identifier
  session_id: 123
  # TSS threshold
  threshold: 2

# Bridge Core connector configuration
core_connector:
  # Core connection settings
  connection:
    # Core RPC endpoint for queries and transactions
    addr: "rpc"
    # Whether to enable TLS connection or not
    enable_tls: false
  # General Core settings
  settings:
    chain_id: "00000"
    denom: "denom"
    min_gas_price: 0

# Bridge Core event subscriber configuration
subscriber:
  # Bridge Core node Tendermint RPC endpoint
  addr: "tcp"
```

Example configuration file can be found [here](./../examples/config/config.example.yaml).

## Vault configuration

[HashiCorp Vault](https://www.vaultproject.io/) is used to store the most sensitive data like keys, private TSS key shares etc.
It should be properly configured and started before running the TSS service.

### Environment variables
To configure the Vault credential for the TSS service, the following environment variables should be set:
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

### Required secrets
The following secrets should be preconfigured in the Vault before running the TSS service:
- local party's Cosmos account private key (use `tss-svc helpers vault set cosmos-account [private_key]` command to set the key);
- local party's self-signed TLS certificate (use `tss-svc helpers vault set tls-cert [path-to-cert] [path-to-key]` command to set the certificate);

All other secrets will be generated and saved automatically during the TSS service launch.
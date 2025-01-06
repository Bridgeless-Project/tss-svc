# P2P module

## Description
Peer-to-peer (P2P) module that contains the core logic for the peer-to-peer communication between the signing TSS nodes (parties) in the network.

## Components
To be added

--- 

## Broadcaster

### Description
P2P broadcaster is responsible for broadcasting messages to all connected peers.
It receives a list of peers to begin broadcasting messages to.
It also can be used to broadcast messages to a specific set of peers.

---

## Connection manager

### Description
P2P connection manager is responsible for managing the peer-to-peer connections and their states.
It holds grpc-connections for each peer and monitors their states.
Different parts of the system can request a list of successfully-connected peers.

A successful connection is a connection that has been established by checking the peer public key and a service mode match.

As the party server can be run in TLS enabled/disabled mode, the connection manager should be able to handle both cases.
In case of a TLS-enabled party server, the connection manager should configure clients with the TLS certificates.
Otherwise, no additional configuration is required.

### Inputs
Manager accepts: 
- the list of peers to connect to;
- current service mode to identify ready-to-serve peers;
- client TLS certificate to identify itself to other peers (optional, in case of TLS-enabled mode).

### Outputs
Manager provides:
- a list of successfully-connected peers;
- a grpc-connection to a specific peer by its public key;
- an option to subscribe to the parties' connection state changes.

--- 

## Party server

### Description
P2P party server is responsible for handling incoming connections from other peers.
To see the API specification and available methods, check the 
- [OpenAPI/Swagger specs](../../api/README.md);
- [Protocol Buffer definitions](../../proto/README.md).

### TLS enabled/disabled modes
As the TSS protocol requires a secure connection between the parties, the server should be able to handle both TLS-enabled and disabled modes.

In case of a TLS-disabled party server, the server should be able to accept incoming connections without any additional configuration.
To identify the peer, server will use the public key from the peer's request.

**NOTE: do not use the TLS-disabled mode in production environments as everyone can use someones' public key to connect to the party.**

In case of a TLS-enabled party server, the server should be configured with the TLS certificates.
It includes:
- server certificate;
- server private key;
- pool of CA certificates to verify the party certificates.
- party's public keys to identify the peers.


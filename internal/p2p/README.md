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

### Inputs
Manager accepts: 
- the list of peers to connect to;
- current service mode to identify ready-to-serve peers;

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

## Messages transport

To ensure the secure and reliable message transport, the P2P module uses the gRPC protocol and the mTLS encryption.
It maps provided parties' certificates to their Bridge Core addresses for authentication and authorization purposes.



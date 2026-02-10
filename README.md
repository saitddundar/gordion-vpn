#  Gordion VPN

Decentralized VPN built with microservices architecture.

##  Architecture

- **Control Plane (Microservices):**
  - Identity Service: Node authentication & key management
  - Discovery Service: Peer discovery & matching
  - Config Service: Network configuration & IP allocation

- **Data Plane:**
  - Agent: VPN client/relay node (P2P + WireGuard)

## Project Structure

```
gordion-vpn/
├── services/       # Microservices
├── pkg/            # Shared libraries
├── api/            # Protocol definitions
├── deployments/    # Docker & K8s
└── docs/           # Documentation
```

## Development Status

Sprint 1: Foundation (in progress)

## TODO

- [ ] Proto definitions
- [ ] Identity service
- [ ] Discovery service
- [ ] Config service
- [ ] Agent implementation

## Tech Stack

- Go 1.21+
- gRPC + Protocol Buffers
- PostgreSQL, Redis
- libp2p, WireGuard
- Docker

## License

MIT

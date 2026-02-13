# 🔐 Gordion VPN

A decentralized, peer-to-peer VPN built with microservices architecture and WireGuard.

## 🎯 Overview

Gordion VPN is a next-generation decentralized VPN that turns users into both clients and relay nodes. Built with a modern microservices architecture, it provides secure, scalable, and distributed VPN connectivity.

## 🏗️ Architecture

### Control Plane (Microservices)

- **Identity Service** ✅ - Node authentication & JWT-based key management
- **Discovery Service** 🔄 - Peer discovery & matching (Coming soon)
- **Config Service** 🔄 - Network configuration & IP allocation (Coming soon)

### Data Plane

- **Agent** 🔄 - VPN client/relay node using libp2p and WireGuard (Coming soon)

### Monitoring Stack

- **Prometheus** ✅ - Metrics collection
- **Grafana** ✅ - Metrics visualization

## 📁 Project Structure

```
gordion-vpn/
├── services/              # Microservices
│   └── identity/          # Identity service ✅
│       ├── cmd/           # Entry points
│       ├── internal/      # Service logic
│       ├── migrations/    # Database migrations
│       └── test/          # Integration tests
├── pkg/                   # Shared libraries
│   ├── logger/            # Structured logging
│   ├── config/            # Configuration management
│   ├── metrics/           # Prometheus metrics
│   ├── grpcutil/          # gRPC utilities
│   └── proto/             # Generated proto code
├── api/proto/             # Protocol definitions
│   ├── identity/v1/       # Identity service API
│   ├── discovery/v1/      # Discovery service API
│   └── config/v1/         # Config service API
├── deployments/           # Docker & K8s configs
│   ├── docker-compose.dev.yml
│   └── prometheus.yml
├── configs/               # Service configurations
│   └── identity.dev.yaml
└── scripts/               # Build & utility scripts
    └── proto-gen.ps1      # Proto code generation
```

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+ (via Docker)
- Protocol Buffers compiler (`protoc`)

### Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/saitddundar/gordion-vpn.git
   cd gordion-vpn
   ```

2. **Start infrastructure (PostgreSQL, Redis, Prometheus, Grafana):**
   ```bash
   docker-compose -f deployments/docker-compose.dev.yml up -d
   ```

3. **Run database migrations:**
   ```bash
   # Connect to PostgreSQL
   docker exec -i gordion-postgres psql -U gordion -d gordion < services/identity/migrations/0001_initial.sql
   ```

4. **Start Identity Service:**
   ```bash
   cd services/identity
   go run ./cmd/server
   ```

5. **Verify service is running:**
   ```bash
   # Check gRPC endpoint
   curl http://localhost:8001

   # Check metrics endpoint
   curl http://localhost:9090/metrics
   ```

## 🧪 Testing

### Run Integration Tests

```bash
cd services/identity
go test -v ./test
```

Expected output:
```
=== RUN   TestIdentityService
=== RUN   TestIdentityService/RegisterNode
    ✓ Node registered successfully!
    ✓ Token validated!
    ✓ Public key retrieved!
--- PASS: TestIdentityService (0.05s)
PASS
```

## 📊 Monitoring

### Access Grafana

1. Open browser: `http://localhost:3000`
2. Login: `admin` / `admin`
3. Add Prometheus data source: `http://prometheus:9090`

### Available Metrics

- **gRPC Requests:** `gordion_grpc_requests_total`
- **Request Duration:** `gordion_grpc_request_duration_seconds`
- **Active Connections:** `gordion_active_connections`
- **Database Queries:** `gordion_db_queries_total`
- **Query Duration:** `gordion_db_query_duration_seconds`

## 🛠️ Development

### Generate Proto Code

```powershell
.\scripts\proto-gen.ps1
```

### Build Services

```bash
# Identity Service
cd services/identity
go build -o identity-server.exe ./cmd/server
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -v ./test

# With coverage
go test -cover ./...
```

## 📦 Tech Stack

### Backend
- **Language:** Go 1.21+
- **RPC Framework:** gRPC + Protocol Buffers
- **Database:** PostgreSQL 15
- **Cache:** Redis 7
- **Logging:** zerolog (structured logging)
- **Metrics:** Prometheus

### Infrastructure
- **Containerization:** Docker
- **Orchestration:** Docker Compose (K8s coming soon)
- **Monitoring:** Prometheus + Grafana

### Networking (Planned)
- **P2P:** libp2p
- **VPN:** WireGuard

## 📈 Development Status

### Sprint 1: Foundation ✅ (100%)
- [x] Monorepo setup
- [x] Proto definitions
- [x] Shared packages (logger, config, grpcutil)
- [x] Proto code generation

### Sprint 2: Identity Service ✅ (100%)
- [x] Database schema & migrations
- [x] Storage layer (PostgreSQL)
- [x] Service layer (JWT authentication)
- [x] gRPC handler
- [x] Integration tests
- [x] Prometheus metrics

### Sprint 3: Discovery Service 🔄 (0%)
- [ ] Redis-based peer registry
- [ ] Peer discovery & matching
- [ ] Heartbeat mechanism
- [ ] Health checks

### Sprint 4: Config Service 🔄 (0%)
- [ ] IP allocation (DHCP-like)
- [ ] Network topology management
- [ ] Configuration distribution

### Sprint 5: Agent (VPN Client) 🔄 (0%)
- [ ] libp2p integration
- [ ] WireGuard tunnel management
- [ ] Peer-to-peer networking
- [ ] Traffic routing

## 🔒 Security Features

- **JWT-based authentication** - Secure token-based auth
- **WireGuard encryption** (Planned) - State-of-the-art VPN protocol
- **Public key infrastructure** - Cryptographic node identity
- **Token expiration** - Automatic token lifecycle management

## 📝 API Documentation

### Identity Service

**Endpoints:**
- `RegisterNode(PublicKey, Version) -> (NodeID, Token, ExpiresAt)`
- `ValidateToken(Token) -> (Valid, NodeID)`
- `GetPublicKey(NodeID) -> (PublicKey)`

**Example Usage:**
```go
// Register a new node
resp, err := client.RegisterNode(ctx, &identityv1.RegisterNodeRequest{
    PublicKey: "your-wireguard-public-key",
    Version:   "1.0.0",
})

// Validate token
validateResp, err := client.ValidateToken(ctx, &identityv1.ValidateTokenRequest{
    Token: resp.Token,
})
```

## 🤝 Contributing

Contributions are welcome! This is an educational project focused on learning microservices, P2P networking, and VPN technologies.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

## 🔗 Links

- [GitHub Repository](https://github.com/saitddundar/gordion-vpn)
- [Protocol Buffers](https://protobuf.dev/)
- [gRPC](https://grpc.io/)
- [WireGuard](https://www.wireguard.com/)

---

**Status:** 🚧 Active Development - Identity Service Complete, Discovery Service In Progress

**Last Updated:** February 2026

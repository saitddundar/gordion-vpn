# Gordion VPN

A decentralized, peer-to-peer VPN built with microservices architecture and WireGuard.

## Overview

Gordion VPN is a decentralized VPN that turns users into both clients and relay nodes. Built with a modern microservices architecture, it provides secure, scalable, and distributed VPN connectivity.

## Architecture

### Control Plane (Microservices)

| Service | Status | Description |
|---------|--------|-------------|
| **Identity Service** | Complete | Node authentication, JWT token management |
| **Discovery Service** | Complete | Peer discovery, registration, heartbeat |
| **Config Service** | Planned | Network configuration, IP allocation |

### Data Plane

| Component | Status | Description |
|-----------|--------|-------------|
| **Agent** | Planned | VPN client/relay node (libp2p + WireGuard) |

### Monitoring

| Tool | Status | Description |
|------|--------|-------------|
| **Prometheus** | Complete | Metrics collection |
| **Grafana** | Complete | Metrics visualization |

## Project Structure

```
gordion-vpn/
├── services/
│   ├── identity/              # Identity service
│   │   ├── cmd/server/        # Entry point
│   │   ├── internal/          # Config, storage, service, gRPC handler
│   │   ├── migrations/        # Database migrations
│   │   └── test/              # Integration tests
│   └── discovery/             # Discovery service
│       ├── cmd/server/        # Entry point
│       ├── internal/          # Config, registry, matcher, gRPC handler
│       └── test/              # Integration tests
├── pkg/
│   ├── logger/                # Structured logging (zerolog)
│   ├── config/                # Configuration management
│   ├── metrics/               # Prometheus metrics
│   ├── grpcutil/              # gRPC error utilities
│   └── proto/                 # Generated protobuf code
├── api/proto/                 # Protocol Buffer definitions
│   ├── identity/v1/
│   ├── discovery/v1/
│   └── config/v1/
├── deployments/               # Docker Compose, Prometheus config
├── configs/                   # Service configuration files
└── scripts/                   # Build and utility scripts
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Protocol Buffers compiler (`protoc`)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/saitddundar/gordion-vpn.git
cd gordion-vpn

# Start infrastructure
docker-compose -f deployments/docker-compose.dev.yml up -d

# Run database migrations
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0001_initial.sql

# Start Identity Service
cd services/identity
go run ./cmd/server

# Start Discovery Service (separate terminal)
cd services/discovery
go run ./cmd/server
```

### Verify Services

```bash
# Identity Service - gRPC on :8001, metrics on :9090
curl http://localhost:9090/metrics

# Discovery Service - gRPC on :8002
```

## Testing

```bash
# Identity Service integration tests
cd services/identity
go test -v -count=1 ./test/...

# Discovery Service integration tests
cd services/discovery
go test -v -count=1 ./test/...
```

## Monitoring

| Endpoint | URL | Credentials |
|----------|-----|-------------|
| Grafana | `http://localhost:3000` | admin / admin |
| Prometheus | `http://localhost:9091` | - |

### Available Metrics

| Metric | Description |
|--------|-------------|
| `gordion_grpc_requests_total` | Total gRPC request count |
| `gordion_grpc_request_duration_seconds` | Request latency histogram |
| `gordion_active_connections` | Current active connections |
| `gordion_db_queries_total` | Database query count |
| `gordion_db_query_duration_seconds` | Database query latency |

## API Reference

### Identity Service (port 8001)

| Method | Request | Response |
|--------|---------|----------|
| `RegisterNode` | `public_key`, `version` | `node_id`, `token`, `expires_at` |
| `ValidateToken` | `token` | `valid`, `node_id` |
| `GetPublicKey` | `node_id` | `public_key` |

### Discovery Service (port 8002)

| Method | Request | Response |
|--------|---------|----------|
| `RegisterPeer` | `token`, `ip_address`, `port`, `region` | `success`, `message` |
| `ListPeers` | `region`, `limit` | `peers[]` |
| `Heartbeat` | `token`, `bandwidth` | `success`, `ttl` |

## Development

### Generate Proto Code

```powershell
.\scripts\proto-gen.ps1
```

### Build Services

```bash
# Identity Service
cd services/identity
go build -o identity-server.exe ./cmd/server

# Discovery Service
cd services/discovery
go build -o discovery-server.exe ./cmd/server
```

## Tech Stack

| Category | Technology |
|----------|-----------|
| Language | Go 1.21+ |
| RPC | gRPC, Protocol Buffers |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Logging | zerolog |
| Metrics | Prometheus, Grafana |
| Containers | Docker, Docker Compose |
| Networking (planned) | libp2p, WireGuard |

## Development Status

### Sprint 1: Foundation - Complete
- Monorepo setup, proto definitions, shared packages

### Sprint 2: Identity Service - Complete
- PostgreSQL storage, JWT authentication, gRPC API, integration tests, Prometheus metrics

### Sprint 3: Discovery Service - Complete
- Redis registry, peer matching, heartbeat mechanism, gRPC API, integration tests

### Sprint 4: Config Service - Planned
- IP allocation, network topology, configuration distribution

### Sprint 5: Agent - Planned
- libp2p integration, WireGuard tunnels, peer-to-peer networking

## License

MIT License - see [LICENSE](LICENSE) file for details.

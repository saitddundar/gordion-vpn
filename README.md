# Gordion VPN

A decentralized, peer-to-peer VPN built with microservices architecture and WireGuard.

## Overview

Gordion VPN is a decentralized VPN that turns users into both clients and relay nodes. Built with a modern microservices architecture, it provides secure, scalable, and distributed VPN connectivity.

## Architecture

### Control Plane (Microservices)

| Service | Port | Status | Description |
|---------|------|--------|-------------|
| **Identity Service** | 8001 | Complete | Node authentication, JWT token management |
| **Discovery Service** | 8002 | Complete | Peer discovery, registration, heartbeat |
| **Config Service** | 8003 | Complete | Network configuration, IP allocation |

### Data Plane

| Component | Status | Description |
|-----------|--------|-------------|
| **Agent** | Planned | VPN client/relay node (libp2p + WireGuard) |

### Monitoring

| Tool | Port | Status | Description |
|------|------|--------|-------------|
| **Prometheus** | 9091 | Complete | Metrics collection from all services |
| **Grafana** | 3000 | Complete | Metrics visualization and dashboards |

## Project Structure

```
gordion-vpn/
├── services/
│   ├── identity/              # Identity service (PostgreSQL)
│   │   ├── cmd/server/        # Entry point
│   │   ├── internal/          # Config, storage, service, gRPC handler
│   │   ├── migrations/        # Database migrations
│   │   └── test/              # Integration tests
│   ├── discovery/             # Discovery service (Redis)
│   │   ├── cmd/server/        # Entry point
│   │   ├── internal/          # Config, registry, matcher, gRPC handler
│   │   └── test/              # Integration tests
│   └── config/                # Config service (Redis)
│       ├── cmd/server/        # Entry point
│       ├── internal/          # Config, allocator, gRPC handler
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
├── scripts/                   # Build and utility scripts
└── Makefile                   # Build automation
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Protocol Buffers compiler (`protoc`)
- Make (optional, for build automation)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/saitddundar/gordion-vpn.git
cd gordion-vpn

# Start infrastructure (PostgreSQL, Redis, Prometheus, Grafana)
make docker-up

# Run database migrations
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0001_initial.sql

# Build all services
make build-all

# Start services (each in a separate terminal)
cd services/identity  && ./identity-server.exe
cd services/discovery && ./discovery-server.exe
cd services/config    && ./config-server.exe
```

### Verify Services

```bash
# Identity Service metrics
curl http://localhost:9090/metrics

# Discovery Service metrics
curl http://localhost:9091/metrics

# Config Service metrics
curl http://localhost:9092/metrics
```

## Makefile Commands

```bash
make help              # Show all available commands

# Build
make build-all         # Build all services
make build-identity    # Build identity service
make build-discovery   # Build discovery service
make build-config      # Build config service

# Test
make test-all          # Run all integration tests
make test-identity     # Run identity tests
make test-discovery    # Run discovery tests
make test-config       # Run config tests

# Docker
make docker-up         # Start infrastructure
make docker-down       # Stop infrastructure
make docker-restart    # Restart infrastructure

# Utilities
make tidy-all          # Run go mod tidy on all modules
make proto             # Generate protobuf code
make clean             # Remove build artifacts
```

## Testing

```bash
# Run all tests
make test-all

# Or individually
cd services/identity  && go test -v -count=1 ./test/...
cd services/discovery && go test -v -count=1 ./test/...
cd services/config    && go test -v -count=1 ./test/...
```

## Monitoring

All services expose Prometheus metrics via dedicated HTTP endpoints.

| Service | Metrics Endpoint |
|---------|-----------------|
| Identity | `http://localhost:9090/metrics` |
| Discovery | `http://localhost:9091/metrics` |
| Config | `http://localhost:9092/metrics` |
| Grafana | `http://localhost:3000` (admin / admin) |
| Prometheus | `http://localhost:9091` |

### Available Metrics

| Metric | Description |
|--------|-------------|
| `gordion_grpc_requests_total` | Total gRPC request count by service, method, status |
| `gordion_grpc_request_duration_seconds` | Request latency histogram |
| `gordion_active_connections` | Current active connections per service |
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

### Config Service (port 8003)

| Method | Request | Response |
|--------|---------|----------|
| `GetConfig` | `token` | `network_cidr`, `mtu`, `dns_servers` |
| `RequestIP` | `token`, `node_id` | `ip_address`, `subnet_mask`, `gateway` |
| `ReleaseIP` | `token`, `node_id`, `ip_address` | `success`, `message` |

## Tech Stack

| Category | Technology |
|----------|-----------|
| Language | Go 1.21+ |
| RPC | gRPC, Protocol Buffers |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Logging | zerolog |
| Metrics | Prometheus, Grafana |
| Build | Make |
| Containers | Docker, Docker Compose |
| Networking (planned) | libp2p, WireGuard |

## Development Status

### Sprint 1: Foundation - Complete
- Monorepo setup, proto definitions, shared packages

### Sprint 2: Identity Service - Complete
- PostgreSQL storage, JWT authentication, gRPC API, integration tests, Prometheus metrics

### Sprint 3: Discovery Service - Complete
- Redis registry, peer matching, heartbeat mechanism, gRPC API, integration tests, Prometheus metrics

### Sprint 4: Config Service - Complete
- IP allocation (DHCP-like), network configuration, gRPC API, integration tests, Prometheus metrics

### Sprint 5: Agent - Next
- libp2p integration, WireGuard tunnels, peer-to-peer networking

## License

MIT License - see [LICENSE](LICENSE) file for details.

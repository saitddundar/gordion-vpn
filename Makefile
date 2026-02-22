.PHONY: build-all build-identity build-discovery build-config \
       test-all test-identity test-discovery test-config \
       docker-up docker-down docker-restart docker-build \
       tidy-all proto clean help

# === BUILD ===

build-all: build-identity build-discovery build-config
	@echo "All services built successfully"

build-identity:
	@echo "Building Identity Service..."
	cd services/identity && go build -o identity-server.exe ./cmd/server

build-discovery:
	@echo "Building Discovery Service..."
	cd services/discovery && go build -o discovery-server.exe ./cmd/server

build-config:
	@echo "Building Config Service..."
	cd services/config && go build -o config-server.exe ./cmd/server

# === TEST ===

test-all: test-identity test-discovery test-config
	@echo "All tests completed"

test-identity:
	@echo "Testing Identity Service..."
	cd services/identity && go test -v -count=1 ./test/...

test-discovery:
	@echo "Testing Discovery Service..."
	cd services/discovery && go test -v -count=1 ./test/...

test-config:
	@echo "Testing Config Service..."
	cd services/config && go test -v -count=1 ./test/...

# === DOCKER ===

docker-up:
	@echo "Starting infrastructure..."
	docker-compose -f deployments/docker-compose.dev.yml up -d

docker-down:
	@echo "Stopping infrastructure..."
	docker-compose -f deployments/docker-compose.dev.yml down

docker-restart: docker-down docker-up

docker-build:
	@echo "Building Docker images..."
	docker-compose -f deployments/docker-compose.dev.yml build

# === DEPENDENCIES ===

tidy-all:
	@echo "Tidying all modules..."
	cd services/identity && go mod tidy
	cd services/discovery && go mod tidy
	cd services/config && go mod tidy

# === PROTO ===

proto:
	@echo "Generating proto code..."
	powershell -File scripts/proto-gen.ps1

# === CLEAN ===

clean:
	@echo "Cleaning build artifacts..."
	del /Q services\identity\identity-server.exe 2>nul || true
	del /Q services\discovery\discovery-server.exe 2>nul || true
	del /Q services\config\config-server.exe 2>nul || true

# === HELP ===

help:
	@echo.
	@echo   Gordion VPN - Available Commands
	@echo   ================================
	@echo.
	@echo   Build:
	@echo     make build-all         Build all services
	@echo     make build-identity    Build identity service
	@echo     make build-discovery   Build discovery service
	@echo     make build-config      Build config service
	@echo.
	@echo   Test:
	@echo     make test-all          Run all tests
	@echo     make test-identity     Run identity tests
	@echo     make test-discovery    Run discovery tests
	@echo     make test-config       Run config tests
	@echo.
	@echo   Docker:
	@echo     make docker-up         Start all containers
	@echo     make docker-down       Stop all containers
	@echo     make docker-restart    Restart all containers
	@echo     make docker-build      Build service Docker images
	@echo.
	@echo   Other:
	@echo     make tidy-all          Run go mod tidy on all modules
	@echo     make proto             Generate protobuf code
	@echo     make clean             Remove build artifacts
	@echo     make help              Show this help
	@echo.

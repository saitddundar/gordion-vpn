# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy workspace files
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY services/discovery/ services/discovery/

# Build
RUN cd services/discovery && go build -o /discovery-server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /discovery-server .
COPY configs/discovery.dev.yaml /app/configs/discovery.dev.yaml

EXPOSE 8002 9091

CMD ["./discovery-server"]

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy workspace files
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY services/identity/ services/identity/

# Build
RUN cd services/identity && go build -o /identity-server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /identity-server .
COPY configs/identity.dev.yaml /app/configs/identity.dev.yaml

EXPOSE 8001 9090

CMD ["./identity-server"]

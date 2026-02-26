# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy workspace files
COPY . .

# Build
RUN cd services/config && go build -o /config-server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /config-server .
COPY configs/config.dev.yaml /app/configs/config.dev.yaml

EXPOSE 8003 9092

CMD ["./config-server"]

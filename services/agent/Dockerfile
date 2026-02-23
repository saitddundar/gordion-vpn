FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY pkg/ pkg/
COPY services/agent/ services/agent/

RUN cd services/agent && go build -o /agent ./cmd/agent

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /agent /usr/local/bin/agent
COPY configs/agent.dev.yaml /etc/gordion/agent.yaml

ENTRYPOINT ["agent"]

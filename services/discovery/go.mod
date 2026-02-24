module github.com/saitddundar/gordion-vpn/services/discovery

go 1.25.4

require (
	github.com/prometheus/client_golang v1.23.2
	github.com/redis/go-redis/v9 v9.18.0
	github.com/saitddundar/gordion-vpn v0.0.0
	github.com/saitddundar/gordion-vpn/pkg/logger v0.0.0
	github.com/saitddundar/gordion-vpn/pkg/metrics v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.79.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/saitddundar/gordion-vpn => ../../
	github.com/saitddundar/gordion-vpn/pkg/logger => ../../pkg/logger
	github.com/saitddundar/gordion-vpn/pkg/metrics => ../../pkg/metrics
)

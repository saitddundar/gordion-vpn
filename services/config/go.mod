module github.com/saitddundar/gordion-vpn/services/config

go 1.25.4

require (
	github.com/redis/go-redis/v9 v9.7.3
	github.com/saitddundar/gordion-vpn v0.0.0
	github.com/saitddundar/gordion-vpn/pkg/logger v0.0.0
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/saitddundar/gordion-vpn => ../../
	github.com/saitddundar/gordion-vpn/pkg/logger => ../../pkg/logger
)

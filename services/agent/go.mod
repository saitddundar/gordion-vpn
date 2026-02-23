module github.com/saitddundar/gordion-vpn/services/agent

go 1.25.4

require (
	github.com/saitddundar/gordion-vpn v0.0.0
	github.com/saitddundar/gordion-vpn/pkg/logger v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.44.0
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
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

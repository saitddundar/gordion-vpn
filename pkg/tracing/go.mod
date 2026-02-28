module github.com/saitddundar/gordion-vpn/pkg/tracing

go 1.24.0

require (
	github.com/saitddundar/gordion-vpn/pkg/logger v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.79.1
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/saitddundar/gordion-vpn/pkg/logger => ../logger

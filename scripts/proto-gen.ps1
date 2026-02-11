# Proto code generation script for Windows

$ErrorActionPreference = "Stop"

# Colors
$Green = "Green"
$Red = "Red"
$Yellow = "Yellow"

Write-Host "Generating Go code from proto files..." -ForegroundColor $Green

# Proto root
$PROTO_ROOT = "$PSScriptRoot\..\api\proto"
$OUT_DIR = "$PSScriptRoot\..\pkg\proto"

# Protoc binary
$PROTOC = "$PSScriptRoot\..\tools\protoc\bin\protoc.exe"

# Check if protoc exists
if (-not (Test-Path $PROTOC)) {
    Write-Host " protoc not found at: $PROTOC" -ForegroundColor $Red
    exit 1
}

# Create output directory
New-Item -ItemType Directory -Force -Path $OUT_DIR | Out-Null

# Generate for each service
$services = @("identity", "discovery", "config")

foreach ($service in $services) {
    Write-Host "Generating $service..." -ForegroundColor $Yellow
    
    $protoFile = "$PROTO_ROOT\$service\v1\$service.proto"
    
    if (-not (Test-Path $protoFile)) {
        Write-Host "Proto file not found: $protoFile" -ForegroundColor $Yellow
        continue
    }
    
    # Run protoc
    & $PROTOC `
        --proto_path="$PROTO_ROOT" `
        --go_out="$PSScriptRoot\.." `
        --go_opt=module=github.com/saitddundar/gordion-vpn `
        --go-grpc_out="$PSScriptRoot\.." `
        --go-grpc_opt=module=github.com/saitddundar/gordion-vpn `
        "$service/v1/$service.proto"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "$service generated successfully" -ForegroundColor $Green
    } else {
        Write-Host "[X] Failed to generate $service" -ForegroundColor $Red
        exit 1
    }
}

Write-Host "`n All proto files generated successfully!" -ForegroundColor $Green
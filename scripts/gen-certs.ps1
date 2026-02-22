# Generate self-signed TLS certificates for development
# Usage: .\scripts\gen-certs.ps1

$certsDir = "certs"

# Create certs directory
if (-Not (Test-Path $certsDir)) {
    New-Item -ItemType Directory -Path $certsDir | Out-Null
}

Write-Host "Generating CA certificate..."
openssl req -x509 -newkey rsa:4096 -days 365 -nodes `
    -keyout "$certsDir/ca-key.pem" `
    -out "$certsDir/ca-cert.pem" `
    -subj "/C=TR/ST=Istanbul/L=Istanbul/O=Gordion VPN/CN=Gordion CA"

Write-Host "Generating server certificate..."
openssl req -newkey rsa:4096 -nodes `
    -keyout "$certsDir/server-key.pem" `
    -out "$certsDir/server-req.pem" `
    -subj "/C=TR/ST=Istanbul/L=Istanbul/O=Gordion VPN/CN=localhost"

# Create extensions file for SAN
$extFile = "$certsDir/server-ext.cnf"
@"
[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = gordion-identity
DNS.3 = gordion-discovery
DNS.4 = gordion-config
IP.1 = 127.0.0.1
IP.2 = 0.0.0.0
"@ | Out-File -FilePath $extFile -Encoding ascii

openssl x509 -req -in "$certsDir/server-req.pem" `
    -CA "$certsDir/ca-cert.pem" `
    -CAkey "$certsDir/ca-key.pem" `
    -CAcreateserial `
    -out "$certsDir/server-cert.pem" `
    -days 365 `
    -extensions v3_req `
    -extfile $extFile

# Cleanup
Remove-Item "$certsDir/server-req.pem" -ErrorAction SilentlyContinue
Remove-Item "$certsDir/server-ext.cnf" -ErrorAction SilentlyContinue
Remove-Item "$certsDir/ca-cert.srl" -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Certificates generated in $certsDir/:"
Write-Host "  ca-cert.pem      - CA certificate (client uses this)"
Write-Host "  ca-key.pem       - CA private key"
Write-Host "  server-cert.pem  - Server certificate"
Write-Host "  server-key.pem   - Server private key"
Write-Host ""
Write-Host "Done!"

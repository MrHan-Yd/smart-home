$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
if (-not (Test-Path .env)) {
  Copy-Item .env.example .env
  Write-Host "created .env from .env.example — please edit DATABASE_URL / REDIS_URL"
}
go run ./cmd/server

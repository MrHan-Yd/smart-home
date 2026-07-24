$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

# 与 PM 对齐：使用 go1.24.3 工具链（go.mod toolchain 行）
$env:GOTOOLCHAIN = "go1.24.3"
$env:CGO_ENABLED = "0"
$env:GOPROXY = if ($env:GOPROXY) { $env:GOPROXY } else { "https://goproxy.cn,direct" }
# 项目内缓存，与全局脏缓存隔离
$env:GOCACHE = Join-Path $PSScriptRoot ".gocache"

New-Item -ItemType Directory -Force -Path $env:GOCACHE, (Join-Path $PSScriptRoot "bin") | Out-Null

if (-not (Test-Path .env)) {
  Copy-Item .env.example .env
  Write-Host "created .env — edit OAUTH_*/DATABASE_URL/REDIS_URL"
}

Write-Host "go env toolchain:" (go env GOTOOLCHAIN GOVERSION 2>$null)
Write-Host "GOCACHE=$($env:GOCACHE)"

& go build -o (Join-Path $PSScriptRoot "bin\server.exe") ./cmd/server
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& (Join-Path $PSScriptRoot "bin\server.exe")

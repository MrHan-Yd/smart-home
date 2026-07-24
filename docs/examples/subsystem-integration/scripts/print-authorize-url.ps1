# 打印授权 URL（Windows）
$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
# 实际路径：scripts 的上一级是 subsystem-integration
$Dir = Split-Path $PSScriptRoot -Parent
$EnvFile = Join-Path $Dir ".env"
if (Test-Path $EnvFile) {
  Get-Content $EnvFile | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -notmatch '=') { return }
    $k, $v = $_.Split('=', 2)
    Set-Item -Path "Env:$($k.Trim())" -Value $v.Trim()
  }
}

$AuthBase = if ($env:AUTH_BASE) { $env:AUTH_BASE } else { "http://127.0.0.1:3000" }
$ClientId = if ($env:CLIENT_ID) { $env:CLIENT_ID } else { "demo_app" }
$Redirect = if ($env:REDIRECT_URI) { $env:REDIRECT_URI } else { "http://127.0.0.1:9999/oauth/callback" }

$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
$verifier = -join ($bytes | ForEach-Object { $_.ToString("x2") })
$sha = [System.Security.Cryptography.SHA256]::Create()
$hash = $sha.ComputeHash([Text.Encoding]::ASCII.GetBytes($verifier))
$challenge = [Convert]::ToBase64String($hash).TrimEnd('=').Replace('+', '-').Replace('/', '_')
$state = -join ((48..57 + 97..102) | Get-Random -Count 16 | ForEach-Object { [char]$_ })

$encRedirect = [uri]::EscapeDataString($Redirect)
$url = "$AuthBase/oauth/authorize?response_type=code&client_id=$ClientId&redirect_uri=$encRedirect&scope=openid%20profile%20email&state=$state&code_challenge=$challenge&code_challenge_method=S256"

Write-Host "CODE_VERIFIER=$verifier"
Write-Host "OAUTH_STATE=$state"
Write-Host ""
Write-Host "浏览器打开:"
Write-Host $url

@echo off
cd /d "%~dp0"

REM 与 PM 统一：go 1.24 + toolchain go1.24.3
set "GOTOOLCHAIN=go1.24.3"
set "CGO_ENABLED=0"
set "GOCACHE=%~dp0.gocache"
set "GOPROXY=https://goproxy.cn,direct"

if not exist ".gocache" mkdir ".gocache"
if not exist "bin" mkdir "bin"

echo [smart-home] downloading/using go1.24.3 if needed...
go version
echo [smart-home] building with toolchain go1.24.3 ...
go build -o bin\server.exe .\cmd\server
if errorlevel 1 (
  echo BUILD FAILED
  pause
  exit /b 1
)

echo [smart-home] starting...
bin\server.exe
pause

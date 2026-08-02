# Port Mortem — One-Command Unmodified Original Test Suite Script (Windows PowerShell)
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "===> Building Go adapter for original picomatch test suite..." -ForegroundColor Cyan
Set-Location "$RootDir\port"
go build -o ../tests/go_adapter.exe ./cmd/adapter

Write-Host "===> Executing unmodified original picomatch test suite..." -ForegroundColor Cyan
$env:NODE_PATH = "$RootDir\original-picomatch\picomatch-master\node_modules"
Set-Location "$RootDir\tests\original"
node "$RootDir\original-picomatch\picomatch-master\node_modules\mocha\bin\mocha.js" "*.js"
Write-Host "===> Original test suite execution COMPLETE!" -ForegroundColor Green

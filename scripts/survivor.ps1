# Port Mortem — One-Command Differential Fuzz Survivor Script (Windows PowerShell)
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "===> Executing Differential Fuzz Survivor Engine (30s)..." -ForegroundColor Cyan
Set-Location "$RootDir\port"
go run ./fuzz_survivor -duration=30s
Write-Host "===> Survivor test run COMPLETE!" -ForegroundColor Green

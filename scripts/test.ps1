# Port Mortem — One-Command Test Script (Windows PowerShell)
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "===> Executing Port Mortem full test suite..." -ForegroundColor Cyan
Set-Location "$RootDir\port"
go test -count=1 ./...
Write-Host "===> All tests PASSED!" -ForegroundColor Green

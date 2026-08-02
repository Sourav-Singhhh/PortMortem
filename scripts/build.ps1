# Port Mortem — One-Command Build Script (Windows PowerShell)
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "===> Building Port Mortem package module..." -ForegroundColor Cyan
Set-Location "$RootDir\port"
go build ./...
Write-Host "===> Build SUCCESSFUL!" -ForegroundColor Green

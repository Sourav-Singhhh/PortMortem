# Port Mortem — One-Command GoDoc Examples Script (Windows PowerShell)
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "===> Executing Port Mortem runnable example tests..." -ForegroundColor Cyan
Set-Location "$RootDir\port"
go test -v -run="^Example" .
Write-Host "===> Example tests PASSED!" -ForegroundColor Green

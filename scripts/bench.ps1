# Port Mortem — One-Command Benchmark Script (Windows PowerShell)
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Write-Host "===> Executing Port Mortem performance benchmarks..." -ForegroundColor Cyan
Set-Location "$RootDir\port"
go test -run="^$" -bench="." -benchmem -count=1 .
Write-Host "===> Benchmark execution COMPLETE!" -ForegroundColor Green

#!/usr/bin/env pwsh
# GoSync 构建脚本

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  GoSync Build" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 代码检查
Write-Host "[1/3] go vet ..." -ForegroundColor Yellow
go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ go vet 失败" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ 通过" -ForegroundColor Green

# 编译服务端
Write-Host "[2/3] 编译 gosync-server.exe ..." -ForegroundColor Yellow
go build -ldflags "-s -w" -o gosync-server.exe ./cmd/server/
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ 服务端编译失败" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ gosync-server.exe" -ForegroundColor Green

# 编译客户端
Write-Host "[3/3] 编译 gosync-client.exe ..." -ForegroundColor Yellow
go build -ldflags "-s -w" -o gosync-client.exe ./cmd/client/
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ 客户端编译失败" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ gosync-client.exe" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  构建完成" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$serverSize = [math]::Round((Get-Item gosync-server.exe).Length / 1MB, 1)
$clientSize = [math]::Round((Get-Item gosync-client.exe).Length / 1MB, 1)
Write-Host "  gosync-server.exe  ${serverSize} MB"
Write-Host "  gosync-client.exe  ${clientSize} MB"
Write-Host ""
Write-Host "  运行服务端:  .\gosync-server.exe"
Write-Host "  运行客户端:  .\gosync-client.exe pull <ip> <path>"

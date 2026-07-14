#!/usr/bin/env pwsh
# GoShare 构建脚本

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  GoShare Build" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 代码检查
Write-Host "[1/2] go vet ..." -ForegroundColor Yellow
go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ go vet 失败" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ 通过" -ForegroundColor Green

# 编译
Write-Host "[2/3] 编译 goshare.exe ..." -ForegroundColor Yellow
go build -ldflags "-s -w" -o goshare.exe ./cmd/goshare/
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ 编译失败" -ForegroundColor Red
    exit 1
}
$beforeSize = [math]::Round((Get-Item goshare.exe).Length / 1MB, 1)
Write-Host "  ✓ goshare.exe  ${beforeSize} MB" -ForegroundColor Green

# UPX 压缩
Write-Host "[3/3] UPX 压缩 ..." -ForegroundColor Yellow
upx --best --lzma goshare.exe
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ UPX 压缩失败" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  构建完成" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$afterSize = [math]::Round((Get-Item goshare.exe).Length / 1MB, 1)
$ratio = [math]::Round(($beforeSize - $afterSize) / $beforeSize * 100, 0)
Write-Host "  压缩前: ${beforeSize} MB  →  压缩后: ${afterSize} MB  (${ratio}%)"
Write-Host ""
Write-Host "  启动服务:  .\goshare.exe serve --root <目录>"
Write-Host "  下载文件:  .\goshare.exe pull <ip> <path>"
Write-Host "  列出目录:  .\goshare.exe list <ip> <path>"

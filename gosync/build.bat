@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ========================================
echo   GoSync Build
echo ========================================
echo.

echo [1/3] go vet ...
go vet ./...
if %ERRORLEVEL% neq 0 (
    echo ✗ go vet 失败
    exit /b 1
)
echo   ✓ 通过

echo [2/3] 编译 gosync-server.exe ...
go build -ldflags="-s -w" -o gosync-server.exe ./cmd/server/
if %ERRORLEVEL% neq 0 (
    echo ✗ 服务端编译失败
    exit /b 1
)
echo   ✓ gosync-server.exe

echo [3/3] 编译 gosync-client.exe ...
go build -ldflags="-s -w" -o gosync-client.exe ./cmd/client/
if %ERRORLEVEL% neq 0 (
    echo ✗ 客户端编译失败
    exit /b 1
)
echo   ✓ gosync-client.exe

echo.
echo ========================================
echo   构建完成
echo ========================================
echo.
echo   服务端:  gosync-server.exe
echo   客户端:  gosync-client.exe pull ^<ip^> ^<path^>

@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ========================================
echo   GoSync Build
echo ========================================
echo.

echo [1/2] go vet ...
go vet ./...
if %ERRORLEVEL% neq 0 (
    echo ✗ go vet 失败
    exit /b 1
)
echo   ✓ 通过

echo [2/3] 编译 gosync.exe ...
go build -ldflags="-s -w" -o gosync.exe ./cmd/gosync/
if %ERRORLEVEL% neq 0 (
    echo ✗ 编译失败
    exit /b 1
)
for %%A in (gosync.exe) do set beforeSize=%%~zA
echo   ✓ gosync.exe

echo [3/3] UPX 压缩 ...
upx --best --lzma gosync.exe
if %ERRORLEVEL% neq 0 (
    echo ✗ UPX 压缩失败
    exit /b 1
)

echo.
echo ========================================
echo   构建完成
echo ========================================
echo.
echo   启动服务:  gosync.exe serve --root ^<目录^>
echo   下载文件:  gosync.exe pull ^<ip^> ^<path^>
echo   列出目录:  gosync.exe list ^<ip^> ^<path^>

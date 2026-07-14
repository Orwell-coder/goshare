#!/usr/bin/env bash
# GoShare build script (Linux/macOS — cross-compiles for Windows)
set -euo pipefail
cd "$(dirname "$0")"

BOLD="\033[1m"
CYAN="\033[36m"
YELLOW="\033[33m"
GREEN="\033[32m"
RED="\033[31m"
RESET="\033[0m"

echo -e "${CYAN}========================================${RESET}"
echo -e "${CYAN}  GoShare Build${RESET}"
echo -e "${CYAN}========================================${RESET}"
echo ""

# ── Step 1: go vet ──────────────────────────────────────────────
echo -e "${YELLOW}[1/3] go vet ...${RESET}"
go vet ./...
echo -e "${GREEN}  ✓ 通过${RESET}"

# ── Step 2: cross-compile for Windows ────────────────────────────
echo -e "${YELLOW}[2/3] 交叉编译 goshare.exe (GOOS=windows GOARCH=amd64) ...${RESET}"
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o goshare.exe ./cmd/goshare/
echo -e "${GREEN}  ✓ goshare.exe${RESET}"

# ── Step 3: UPX compress (optional) ──────────────────────────────
if command -v upx &>/dev/null; then
  echo -e "${YELLOW}[3/3] UPX 压缩 ...${RESET}"
  BEFORE=$(du -h goshare.exe | cut -f1)
  upx --best --lzma goshare.exe
  AFTER=$(du -h goshare.exe | cut -f1)
  echo -e "${GREEN}  ✓ ${BEFORE} → ${AFTER}${RESET}"
else
  echo -e "${YELLOW}[3/3] UPX 未安装，跳过压缩${RESET}"
fi

echo ""
echo -e "${CYAN}========================================${RESET}"
echo -e "${CYAN}  构建完成${RESET}"
echo -e "${CYAN}========================================${RESET}"
echo ""
echo "  goshare.exe  (Windows amd64)"
echo ""
echo "  Windows 上使用:"
echo "    .\\goshare.exe serve --root <目录>"
echo "    .\\goshare.exe pull <ip> <path>"
echo "    .\\goshare.exe list <ip> <path>"

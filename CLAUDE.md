# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```powershell
# Full build with UPX compression (PowerShell — local dev build with go vet + UPX)
.\build.ps1

# Full build with UPX compression (CMD — local dev build with UPX)
build.bat

# Build directly without UPX (skips compression step)
go build -ldflags "-s -w" -o goshare.exe ./cmd/goshare/

# Cross-compile Windows binary from Linux
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o goshare.exe ./cmd/goshare/

# Vet
go vet ./...

# Run server (current directory as root)
.\goshare.exe serve

# Run server with specific roots
.\goshare.exe serve --root D:\data --root E:\work

# Quick smoke test: pull from a running server
.\goshare.exe list 127.0.0.1 /
.\goshare.exe pull 127.0.0.1 /some-dir --output D:\test-out
```

There are no tests in this project. UPX is optional in the CI workflow (falls back to uncompressed binary if UPX fails).

## Installation Options

This project has multiple distribution channels:
- **winget**: `winget install Orwell-coder.goshare` (automated via GitHub Actions)
- **go install**: `go install github.com/Orwell-coder/goshare/cmd/goshare@latest`
- **GitHub Releases**: Pre-compiled binaries with SHA256 checksums

## Architecture

Single-binary Go application (`goshare.exe`) — same binary is both server and client. Module path: `github.com/Orwell-coder/goshare` (Go 1.24+, single dependency: `klauspost/compress` for zstd).

### Dual protocol design

| Protocol | Port  | Purpose                   | Encoding                          |
| -------- | ----- | ------------------------- | --------------------------------- |
| HTTP     | 18080 | Browser browsing + zip DL | `net/http` + streaming zip        |
| TCP      | 19090 | CLI high-speed transfer   | gob control frames + raw binary chunks |

### Package layout

- **`cmd/goshare/`** — CLI entry point. Three subcommands (`serve`, `pull`, `list`) each with their own `flag.FlagSet`. Includes `reorderArgs()` to allow interleaved flags and positional args. The `serve` subcommand starts both an HTTP server and a TCP server in goroutines, sharing a single `transfer.Engine`.

- **`internal/transfer/`** — Core transfer engine. `Engine` is the shared component between HTTP and TCP paths. Key pieces:
  - `ChunkSender` — streams files in 4MB chunks via `sync.Pool` buffer reuse, never loads whole files into memory.
  - `Compressor` — zstd compression with encoder/decoder pools. `isCompressedFormat()` checks magic bytes to skip already-compressed formats (ZIP, JPEG, PNG, MP4, etc.).
  - `sendfile_windows.go` — platform-specific file send using `io.CopyBuffer` (not Windows TransmitFile; uses 1MB pool buffer). Cross-platform would need a parallel `sendfile_linux.go`.

- **`internal/tcp/`** — TCP protocol layer. `Server` accepts connections with TCP_NODELAY + 1MB buffers. `Session` handles the request/response loop: `ListRequest` → directory walk, `DownloadReq` → chunk streaming per file. Root directory resolution supports multiple `--root` paths and remembers `listRoot` for sub-directory context.

- **`internal/http/`** — Browser interface. `handleBrowse` renders directory listings (HTML template), `handleDownload` streams directories as zip (via `archive/zip`, on-the-fly, no temp files). Path resolution uses root directory name as the first path segment.

- **`internal/proto/`** — Wire protocol. Control messages use gob (registered in `init()`), data chunks use raw binary format: `[type:1B][pathLen:2B][path:N][offset:8B][dataLen:4B][data:N][isLast:1B]`. The encoder uses an 8MB bufio buffer to batch chunks without intermediate flushes.

- **`internal/filesvc/`** — File system operations. `Walk()` for recursive directory listing (directories-first sort), `WalkShallow()` for single-level "list" command, `WalkConcurrent()` for SHA256 checksums. `Exists()` compares size + modtime with 1-second tolerance for FAT/exFAT compatibility.

- **`pkg/client/`** — Client library. `Conn` wraps TCP connection with proto encoder/decoder. `Pull()` does: connect → list remote → diff against local files → download batches with progress bar. `Progress` uses Windows Console API (`GetConsoleScreenBufferInfo`) for terminal width detection, with a stub for non-Windows platforms returning 80 columns.

### Protocol flow

```
Client                          Server (TCP Session)
  │                                │
  │──── ListRequest{path} ────────>│  WalkDir / WalkShallow
  │<─── ListResponse{files} ──────│
  │                                │
  │──── DownloadReq{files[]} ─────>│
  │<─── DataChunk × N ────────────│  (per file, streamed in 4MB chunks)
  │<─── BatchDone{path} ──────────│
  │                                │
```

### Key design decisions

- All files stream in 4MB chunks regardless of size — no special "large file" path, no full-file reads into memory.
- Compression is checked per-chunk: if a chunk's magic bytes indicate an already-compressed format, compression is skipped for that chunk.
- Incremental download: client diffs remote listing against local files by size + modtime (1s tolerance for FAT/exFAT), only downloads changed/missing files.
- TCP 8MB write buffer allows two full chunks + headers to batch before a syscall flush.
- Panic recovery on both HTTP handlers and TCP sessions with full stack trace logging.
- Flag parsing supports interleaved flags and positional arguments via `reorderArgs()` (e.g., `pull 127.0.0.1 /data -c 4` works).

### Serve flags

| Flag                  | Short | Default   | Description |
| --------------------- | ----- | --------- | ----------- |
| `--http`            | `-H` | `:18080` | HTTP listen address |
| `--tcp`             | `-T` | `:19090` | TCP listen address |
| `--root`            | `-r` | cwd       | Root directories (repeatable) |
| `--concurrency`     | `-n` | `8`      | Concurrent file transfers |
| `--chunk-size`      | `-k` | `4`      | Chunk size in MB |
| `--large-threshold` | `-l` | `16`     | Large file threshold in MB |
| `--compress`        | —     | `true`   | Enable zstd compression |
| `--no-compress`     | —     | `false`  | Disable zstd compression |
| `--compress-level`  | `-z` | `3`      | zstd level (1=fastest, 22=best) |
| `--rate-limit`      | `-R` | `0`      | Rate limit in MB/s (0=unlimited) |

### Pull flags

| Flag              | Short | Default     | Description |
| ----------------- | ----- | ----------- | ----------- |
| `--port`        | `-p` | `19090`    | TCP port |
| `--output`      | `-o` | remote name | Local output directory |
| `--concurrency` | `-c` | `8`        | Concurrent downloads |

## CI/CD Workflows

- **CI**: Runs `go vet` and cross-compiles on `ubuntu-latest` on push/pr to `main`/`master`
- **Release**: Triggered on `v*` tags. Builds (uncompressed for AV compatibility), generates SHA256 checksums, creates multi-file winget manifests, and publishes a GitHub Release
- **Winget**: Automated submission to `microsoft/winget-pkgs` via PR after release (requires `WINGET_TOKEN` secret)

## Platform

Windows-first with basic non-Windows support via build tags:
- **Windows**: Terminal width detection via `kernel32.dll` `GetConsoleScreenBufferInfo`
- **Linux/macOS**: Falls back to fixed 80-column width. Porting would require:
  - `sendfile_linux.go` using `syscall.Sendfile` for zero-copy
  - `term_linux.go` using `golang.org/x/term` for width detection

## Limitations & Design Constraints

- One-way download only (Server → Client), no upload or bidirectional sync
- LAN-trusted environment, no authentication or encryption
- Windows is the primary target; other platforms work but with reduced terminal UI

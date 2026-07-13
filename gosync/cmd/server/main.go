package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"gosync/internal/http"
	"gosync/internal/tcp"
	"gosync/internal/transfer"
)

func main() {
	httpAddr := ":18080"
	tcpAddr := ":19090"
	var rootDirs []string
	concurrency := 8
	chunkSize := 4
	largeThreshMB := 16
	compress := true
	compressLevel := 3
	rateLimit := 0

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--http", "-H":
			i++
			if i < len(args) {
				httpAddr = args[i]
			}
		case "--tcp", "-T":
			i++
			if i < len(args) {
				tcpAddr = args[i]
			}
		case "--root", "-r":
			i++
			if i < len(args) {
				rootDirs = append(rootDirs, args[i])
			}
		case "--concurrency", "-n":
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil {
					concurrency = n
				}
			}
		case "--chunk-size", "-k":
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil {
					chunkSize = n
				}
			}
		case "--large-threshold", "-l":
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil {
					largeThreshMB = n
				}
			}
		case "--compress":
			compress = true
		case "--no-compress":
			compress = false
		case "--compress-level", "-z":
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil {
					compressLevel = n
				}
			}
		case "--rate-limit", "-R":
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil {
					rateLimit = n
				}
			}
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "未知参数: %s\n\n", arg)
			printHelp()
			os.Exit(1)
		}
	}

	if len(rootDirs) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无法获取当前目录")
			os.Exit(1)
		}
		rootDirs = append(rootDirs, cwd)
		fmt.Printf("未指定 --root，默认使用当前目录: %s\n", cwd)
	}

	// Validate root directories
	for _, dir := range rootDirs {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "错误: 目录不存在或不可访问: %s\n", dir)
			os.Exit(1)
		}
	}

	cfg := transfer.Config{
		Concurrency:    concurrency,
		ChunkSize:      chunkSize * 1024 * 1024,
		LargeThreshold: int64(largeThreshMB) * 1024 * 1024,
		Compression:    compress,
		CompressLevel:  compressLevel,
		RateLimitMBs:   rateLimit,
	}

	engine := transfer.NewEngine(cfg)

	log.Println("========================================")
	log.Println("  GoSync Server")
	log.Println("========================================")
	log.Printf("  HTTP:      http://%s", httpAddr)
	log.Printf("  TCP:       tcp://%s", tcpAddr)
	log.Printf("  根目录:    %s", strings.Join(rootDirs, ", "))
	log.Printf("  并发数:    %d", concurrency)
	log.Printf("  分块大小:  %d MB", chunkSize)
	log.Printf("  大文件阈值: %d MB", largeThreshMB)
	log.Printf("  压缩:      %v (level %d)", compress, compressLevel)
	if rateLimit > 0 {
		log.Printf("  限速:      %d MB/s", rateLimit)
	}
	log.Println("========================================")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	tcpServer := tcp.NewServer(cfg, engine, rootDirs)
	httpServer := http.NewServer(rootDirs)

	errCh := make(chan error, 2)

	go func() {
		errCh <- tcpServer.ListenAndServe(ctx, tcpAddr)
	}()

	go func() {
		errCh <- httpServer.ListenAndServe(ctx, httpAddr)
	}()

	log.Println("服务已启动，按 Ctrl+C 停止")

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("服务器错误: %v", err)
		}
	case <-ctx.Done():
		log.Println("正在关闭...")
	}

	cancel()
	log.Println("服务器已停止")
}

func printHelp() {
	fmt.Println("GoSync Server — 高性能局域网文件同步服务")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  gosync-server --root <目录> [--root <目录2>] [flags]")
	fmt.Println()
	fmt.Println("参数:")
	fmt.Println("  --root,  -r <path>      允许访问的根目录 (可多次指定, 默认当前目录)")
	fmt.Println("  --http,  -H <addr>      HTTP 监听地址 (默认 :18080)")
	fmt.Println("  --tcp,   -T <addr>      TCP 监听地址 (默认 :19090)")
	fmt.Println("  --concurrency, -n <n>   并行传输文件数 (默认 8)")
	fmt.Println("  --chunk-size, -k <mb>   分块大小 MB (默认 4)")
	fmt.Println("  --large-threshold, -l <mb>  大文件阈值 MB (默认 16)")
	fmt.Println("  --compress              启用 zstd 压缩 (默认)")
	fmt.Println("  --no-compress           禁用压缩")
	fmt.Println("  --compress-level, -z <n> zstd 压缩级别 1-22 (默认 3)")
	fmt.Println("  --rate-limit, -R <mb>   限速 MB/s, 0=不限 (默认 0)")
	fmt.Println("  --help,  -h             显示帮助")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  gosync-server --root D:\\data --root E:\\work")
	fmt.Println("  gosync-server -r D:\\share -H :8080 -T :9090 -n 16")
}

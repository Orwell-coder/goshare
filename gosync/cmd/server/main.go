package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"gosync/internal/http"
	"gosync/internal/tcp"
	"gosync/internal/transfer"
)

func main() {
	var (
		httpAddr      string
		tcpAddr       string
		rootDirs      rootFlags
		concurrency   int
		chunkSize     int
		largeThreshMB int
		compress      bool
		compressLevel int
		rateLimit     int
	)

	flag.StringVar(&httpAddr, "http", ":18080", "HTTP 监听地址")
	flag.StringVar(&tcpAddr, "tcp", ":19090", "TCP 监听地址")
	flag.Var(&rootDirs, "root", "允许访问的根目录 (可多次指定)")
	flag.IntVar(&concurrency, "concurrency", 8, "并行传输文件数")
	flag.IntVar(&chunkSize, "chunk-size", 4, "分块大小 (MB)")
	flag.IntVar(&largeThreshMB, "large-threshold", 16, "大文件阈值 (MB)")
	flag.BoolVar(&compress, "compress", true, "启用 zstd 压缩")
	flag.IntVar(&compressLevel, "compress-level", 3, "zstd 压缩级别 (1-22)")
	flag.IntVar(&rateLimit, "rate-limit", 0, "限速 (MB/s), 0=不限速")
	flag.Parse()

	if len(rootDirs) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 至少需要指定一个 --root 目录")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "用法: gosync-server serve --root <dir> [--root <dir2>] [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "示例:")
		fmt.Fprintln(os.Stderr, `  gosync-server serve --root D:\data --root E:\work`)
		fmt.Fprintln(os.Stderr, `  gosync-server serve --root /home/user/data --http :18080 --tcp :19090`)
		flag.PrintDefaults()
		os.Exit(1)
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
	log.Printf("  HTTP:     http://%s", httpAddr)
	log.Printf("  TCP:      tcp://%s", tcpAddr)
	log.Printf("  根目录:   %s", strings.Join(rootDirs, ", "))
	log.Printf("  并发数:   %d", concurrency)
	log.Printf("  分块大小: %d MB", chunkSize)
	log.Printf("  大文件阈值: %d MB", largeThreshMB)
	log.Printf("  压缩:     %v (level %d)", compress, compressLevel)
	if rateLimit > 0 {
		log.Printf("  限速:     %d MB/s", rateLimit)
	}
	log.Println("========================================")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	tcpServer := tcp.NewServer(cfg, engine, rootDirs)
	httpServer := http.NewServer(rootDirs)

	// Start both servers
	errCh := make(chan error, 2)

	go func() {
		errCh <- tcpServer.ListenAndServe(ctx, tcpAddr)
	}()

	go func() {
		errCh <- httpServer.ListenAndServe(ctx, httpAddr)
	}()

	log.Println("服务已启动，按 Ctrl+C 停止")

	// Wait for error or signal
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

// rootFlags is a custom flag type for collecting multiple --root values.
type rootFlags []string

func (f *rootFlags) String() string {
	return strings.Join(*f, ", ")
}

func (f *rootFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/Orwell-coder/goshare/internal/http"
	"github.com/Orwell-coder/goshare/internal/tcp"
	"github.com/Orwell-coder/goshare/internal/transfer"
	"github.com/Orwell-coder/goshare/pkg/client"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// ── serve flags ──────────────────────────────────────────────
	serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)

	var (
		httpAddr      string
		tcpAddr       string
		rootDirs      []string
		concurrency   int
		chunkSizeMB   int
		largeThreshMB int
		compress      bool
		noCompress    bool
		compressLevel int
		rateLimitMBs  int
	)
	serveFlags.StringVar(&httpAddr, "http", ":18080", "HTTP 监听地址")
	serveFlags.StringVar(&httpAddr, "H", ":18080", "HTTP 监听地址（短）")
	serveFlags.StringVar(&tcpAddr, "tcp", ":19090", "TCP 监听地址")
	serveFlags.StringVar(&tcpAddr, "T", ":19090", "TCP 监听地址（短）")
	serveFlags.Func("root", "允许访问的根目录（可多次指定, 默认当前目录）", func(s string) error {
		rootDirs = append(rootDirs, s)
		return nil
	})
	serveFlags.Func("r", "同 --root", func(s string) error {
		rootDirs = append(rootDirs, s)
		return nil
	})
	serveFlags.IntVar(&concurrency, "concurrency", 8, "并行传输文件数")
	serveFlags.IntVar(&concurrency, "n", 8, "并行传输文件数（短）")
	serveFlags.IntVar(&chunkSizeMB, "chunk-size", 4, "分块大小 (MB)")
	serveFlags.IntVar(&chunkSizeMB, "k", 4, "分块大小 (MB)（短）")
	serveFlags.IntVar(&largeThreshMB, "large-threshold", 16, "大文件阈值 (MB)")
	serveFlags.IntVar(&largeThreshMB, "l", 16, "大文件阈值 (MB)（短）")
	serveFlags.BoolVar(&compress, "compress", true, "启用 zstd 压缩（默认）")
	serveFlags.BoolVar(&noCompress, "no-compress", false, "禁用 zstd 压缩")
	serveFlags.IntVar(&compressLevel, "compress-level", 3, "zstd 压缩级别 (1=最快, 22=最优)")
	serveFlags.IntVar(&compressLevel, "z", 3, "zstd 压缩级别（短）")
	serveFlags.IntVar(&rateLimitMBs, "rate-limit", 0, "限速 MB/s (0=不限)")
	serveFlags.IntVar(&rateLimitMBs, "R", 0, "限速 MB/s (0=不限)（短）")

	// ── pull flags ───────────────────────────────────────────────
	pullFlags := flag.NewFlagSet("pull", flag.ExitOnError)
	var (
		pullPort        int
		pullOutput      string
		pullConcurrency int
	)
	pullFlags.IntVar(&pullPort, "port", 19090, "TCP 端口")
	pullFlags.IntVar(&pullPort, "p", 19090, "TCP 端口（短）")
	pullFlags.StringVar(&pullOutput, "output", "", "本地输出目录（默认取远程路径最后一段）")
	pullFlags.StringVar(&pullOutput, "o", "", "本地输出目录（短）")
	pullFlags.IntVar(&pullConcurrency, "concurrency", 8, "并行下载数")
	pullFlags.IntVar(&pullConcurrency, "c", 8, "并行下载数（短）")

	// ── list flags ───────────────────────────────────────────────
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	var listPort int
	listFlags.IntVar(&listPort, "port", 19090, "TCP 端口")
	listFlags.IntVar(&listPort, "p", 19090, "TCP 端口（短）")

	// ── dispatch ─────────────────────────────────────────────────
	switch os.Args[1] {
	case "serve":
		serveFlags.Parse(reorderArgs(os.Args[2:]))
		runServe(serveConfig{
			httpAddr:      httpAddr,
			tcpAddr:       tcpAddr,
			rootDirs:      rootDirs,
			concurrency:   concurrency,
			chunkSizeMB:   chunkSizeMB,
			largeThreshMB: largeThreshMB,
			compress:      compress && !noCompress,
			compressLevel: compressLevel,
			rateLimitMBs:  rateLimitMBs,
		})
	case "pull":
		pullFlags.Parse(reorderArgs(os.Args[2:]))
		args := pullFlags.Args()
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "用法: goshare pull <host> <remote-path> [flags]")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "示例:")
			fmt.Fprintln(os.Stderr, `  goshare pull 192.168.1.100 /data/movies`)
			fmt.Fprintln(os.Stderr, `  goshare pull 192.168.1.100 /data/movies --output D:\downloads`)
			os.Exit(1)
		}
		output := pullOutput
		if output == "" {
			output = filepath.Base(filepath.FromSlash(args[1]))
		}
		if err := client.Pull(client.PullOptions{
			Host:        args[0],
			Port:        pullPort,
			RemotePath:  args[1],
			OutputDir:   output,
			Concurrency: pullConcurrency,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "list":
		listFlags.Parse(reorderArgs(os.Args[2:]))
		args := listFlags.Args()
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "用法: goshare list <host> <remote-path> [flags]")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "示例:")
			fmt.Fprintln(os.Stderr, `  goshare list 192.168.1.100 /data`)
			os.Exit(1)
		}
		if err := client.ListRemote(args[0], listPort, args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// ── serve config & run ───────────────────────────────────────────────

type serveConfig struct {
	httpAddr, tcpAddr       string
	rootDirs                []string
	concurrency, chunkSizeMB, largeThreshMB int
	compress                bool
	compressLevel           int
	rateLimitMBs            int
}

func runServe(cfg serveConfig) {
	if len(cfg.rootDirs) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无法获取当前目录")
			os.Exit(1)
		}
		cfg.rootDirs = append(cfg.rootDirs, cwd)
		fmt.Printf("未指定 --root，默认使用当前目录: %s\n", cwd)
	}
	for _, dir := range cfg.rootDirs {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "错误: 目录不存在或不可访问: %s\n", dir)
			os.Exit(1)
		}
	}

	engine := transfer.NewEngine(transfer.Config{
		Concurrency:    cfg.concurrency,
		ChunkSize:      cfg.chunkSizeMB * 1024 * 1024,
		LargeThreshold: int64(cfg.largeThreshMB) * 1024 * 1024,
		Compression:    cfg.compress,
		CompressLevel:  cfg.compressLevel,
		RateLimitMBs:   cfg.rateLimitMBs,
	})

	log.Println("========================================")
	log.Println("  GoShare Server")
	log.Println("========================================")
	log.Printf("  HTTP:      http://%s", cfg.httpAddr)
	log.Printf("  TCP:       tcp://%s", cfg.tcpAddr)
	log.Printf("  根目录:    %s", strings.Join(cfg.rootDirs, ", "))
	log.Printf("  并发数:    %d", cfg.concurrency)
	log.Printf("  分块大小:  %d MB", cfg.chunkSizeMB)
	log.Printf("  大文件阈值: %d MB", cfg.largeThreshMB)
	log.Printf("  压缩:      %v (level %d)", cfg.compress, cfg.compressLevel)
	if cfg.rateLimitMBs > 0 {
		log.Printf("  限速:      %d MB/s", cfg.rateLimitMBs)
	}
	log.Println("========================================")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	tcpServer := tcp.NewServer(engine.Config(), engine, cfg.rootDirs)
	httpServer := http.NewServer(cfg.rootDirs)

	errCh := make(chan error, 2)

	go func() { errCh <- tcpServer.ListenAndServe(ctx, cfg.tcpAddr) }()
	go func() { errCh <- httpServer.ListenAndServe(ctx, cfg.httpAddr) }()

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

// ── top-level usage ─────────────────────────────────────────────────

func printUsage() {
	fmt.Println("GoShare — 高性能局域网文件分享工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  goshare <command> [arguments]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  serve    启动文件分享服务端")
	fmt.Println("  pull     从服务器下载文件夹")
	fmt.Println("  list     列出远程目录内容")
	fmt.Println("  help     显示帮助")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  goshare serve --root D:\\share")
	fmt.Println("  goshare pull 192.168.1.100 /data/movies --output D:\\downloads")
	fmt.Println("  goshare list 192.168.1.100 /data")
	fmt.Println()
	fmt.Println("运行 'goshare <command> --help' 查看各命令详细用法。")
}

// reorderArgs moves flags (and their values) before positional arguments
// so that flag.FlagSet can parse them when interleaved.
// e.g. "pull 127.0.0.1 /data -c 4" → "-c 4 127.0.0.1 /data"
func reorderArgs(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			positional = append(positional, args[i:]...)
			break
		}
		if strings.HasPrefix(args[i], "-") && !isBareValue(args[i]) {
			flags = append(flags, args[i])
			// Consume the flag's value if the next arg doesn't look like a flag
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}

// isBareValue returns true for strings like "-1" or "-0.5" that look like
// negative numbers, not flags.
func isBareValue(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	return s[1] >= '0' && s[1] <= '9'
}

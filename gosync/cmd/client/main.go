package main

import (
	"fmt"
	"os"
	"strconv"

	"gosync/pkg/client"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "pull":
		pullCmd()
	case "list":
		listCmd()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func parseArgs(args []string) (positional []string, opts map[string]string) {
	opts = make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) >= 2 && arg[0] == '-' {
			name := arg
			if name[1] == '-' {
				name = name[2:]
			} else {
				name = name[1:]
			}
			if i+1 < len(args) && (len(args[i+1]) == 0 || args[i+1][0] != '-') {
				opts[name] = args[i+1]
				i++
			} else {
				opts[name] = "true"
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return
}

func pullCmd() {
	positional, opts := parseArgs(os.Args[2:])

	if len(positional) < 2 {
		fmt.Fprintln(os.Stderr, "用法: gosync-client pull <host> <remote-path> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "示例:")
		fmt.Fprintln(os.Stderr, `  gosync-client pull 192.168.1.100 /data/movies`)
		fmt.Fprintln(os.Stderr, `  gosync-client pull 192.168.1.100 /data/movies --output D:\downloads`)
		os.Exit(1)
	}

	port := 19090
	if v, ok := opts["port"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}
	if v, ok := opts["p"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}

	output := "."
	if v, ok := opts["output"]; ok {
		output = v
	}
	if v, ok := opts["o"]; ok {
		output = v
	}

	concurrency := 8
	if v, ok := opts["concurrency"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			concurrency = n
		}
	}
	if v, ok := opts["c"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			concurrency = n
		}
	}

	pullOpts := client.PullOptions{
		Host:        positional[0],
		Port:        port,
		RemotePath:  positional[1],
		OutputDir:   output,
		Concurrency: concurrency,
	}

	if err := client.Pull(pullOpts); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func listCmd() {
	positional, opts := parseArgs(os.Args[2:])

	if len(positional) < 2 {
		fmt.Fprintln(os.Stderr, "用法: gosync-client list <host> <remote-path> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "示例:")
		fmt.Fprintln(os.Stderr, `  gosync-client list 192.168.1.100 /data`)
		os.Exit(1)
	}

	port := 19090
	if v, ok := opts["port"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}
	if v, ok := opts["p"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}

	if err := client.ListRemote(positional[0], port, positional[1]); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("GoSync Client — 高性能局域网文件同步下载工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  gosync-client pull <host> <remote-path> [flags]   下载文件夹")
	fmt.Println("  gosync-client list <host> <remote-path> [flags]   列出远程目录")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  pull    从服务器下载整个文件夹")
	fmt.Println("  list    列出远程目录内容")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --output, -o string     本地输出目录 (默认当前目录)")
	fmt.Println("  --port, -p int          TCP 端口 (默认 19090)")
	fmt.Println("  --concurrency, -c int   并行下载数 (默认 8)")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  gosync-client pull 192.168.1.100 /data/movies --output D:\\downloads")
	fmt.Println("  gosync-client list 192.168.1.100 /data")
}

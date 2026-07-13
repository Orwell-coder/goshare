package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gosync/internal/filesvc"
	"gosync/internal/proto"
	"gosync/internal/transfer"
)

// PullOptions configures a pull (download) operation.
type PullOptions struct {
	Host        string
	Port        int
	RemotePath  string
	OutputDir   string
	Concurrency int
}

// Pull downloads a directory tree from the server.
func Pull(opts PullOptions) error {
	if opts.Port == 0 {
		opts.Port = 19090
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 4
	}

	// Connect to server
	fmt.Printf("连接到 %s:%d...\n", opts.Host, opts.Port)
	conn, err := Connect(opts.Host, opts.Port)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Printf("已连接到 %s\n", conn.RemoteAddr())

	// Request file listing
	fmt.Printf("获取目录列表: %s\n", opts.RemotePath)
	resp, err := conn.List(opts.RemotePath, 0)
	if err != nil {
		return fmt.Errorf("获取目录列表失败: %w", err)
	}

	fmt.Printf("远程目录: %d 个条目 (%d 文件, %s)\n",
		len(resp.Files),
		filesvc.FileCount(resp.Files),
		formatBytes(filesvc.TotalSize(resp.Files)))

	// Compute diff: which files need to be downloaded
	var toDownload []string
	progress := NewProgress()

	for _, fi := range resp.Files {
		if fi.IsDir {
			// Ensure local directory exists
			localPath := filepath.Join(opts.OutputDir, filepath.FromSlash(fi.Path))
			os.MkdirAll(localPath, 0755)
			continue
		}

		localPath := filepath.Join(opts.OutputDir, filepath.FromSlash(fi.Path))
		if filesvc.Exists(localPath, fi) {
			// File already exists with matching size and mod time
			continue
		}

		toDownload = append(toDownload, fi.Path)
		progress.AddFile(fi.Path, fi.Size)
	}

	if len(toDownload) == 0 {
		fmt.Println("所有文件已是最新，无需下载。")
		return nil
	}

	fmt.Printf("需要下载: %d 个文件\n", len(toDownload))

	// Download files in batches for efficiency
	batchSize := opts.Concurrency * 2
	for i := 0; i < len(toDownload); i += batchSize {
		end := i + batchSize
		if end > len(toDownload) {
			end = len(toDownload)
		}
		batch := toDownload[i:end]

		if err := conn.Download(batch, func(chunk *proto.DataChunk) error {
			if err := transfer.WriteFileChunk(opts.OutputDir, chunk); err != nil {
				return err
			}
			progress.Update(chunk.Path, chunk.Offset+int64(len(chunk.Data)))
			if chunk.IsLast {
				progress.Done(chunk.Path)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("下载失败: %w", err)
		}

		progress.Render()
	}

	progress.Summary()
	return nil
}

// ListRemote fetches and displays the remote directory listing.
func ListRemote(host string, port int, remotePath string) error {
	if port == 0 {
		port = 19090
	}

	fmt.Printf("连接到 %s:%d...\n", host, port)
	conn, err := Connect(host, port)
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := conn.List(remotePath, 1)
	if err != nil {
		return fmt.Errorf("获取列表失败: %w", err)
	}

	fmt.Printf("\n远程目录: %s\n", resp.RootDir)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-50s %10s %s\n", "路径", "大小", "类型")
	fmt.Println(strings.Repeat("-", 80))

	var totalSize int64
	fileCount := 0
	for _, fi := range resp.Files {
		typeStr := "文件"
		if fi.IsDir {
			typeStr = "目录"
		}
		fmt.Printf("%-50s %10s %s\n", fi.Path, formatBytes(fi.Size), typeStr)
		if !fi.IsDir {
			totalSize += fi.Size
			fileCount++
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%d 文件, %d 目录, 共 %s\n",
		fileCount, len(resp.Files)-fileCount, formatBytes(totalSize))
	return nil
}

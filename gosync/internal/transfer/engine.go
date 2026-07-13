package transfer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gosync/internal/filesvc"
	"gosync/internal/proto"
)

// Config holds transfer engine configuration.
type Config struct {
	Concurrency    int
	ChunkSize      int
	LargeThreshold int64
	Compression    bool
	CompressLevel  int
	RateLimitMBs   int // 0 = unlimited
}

// DefaultConfig returns sensible defaults for a gigabit LAN.
func DefaultConfig() Config {
	return Config{
		Concurrency:    8,
		ChunkSize:      DefaultChunkSize,
		LargeThreshold: 0, // deprecated: all files now stream in chunks
		Compression:    true,
		CompressLevel:  3,
		RateLimitMBs:   0,
	}
}

// Engine is the core transfer engine shared by HTTP and TCP handlers.
type Engine struct {
	cfg        Config
	compressor *Compressor
}

// NewEngine creates a transfer engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{
		cfg:        cfg,
		compressor: NewCompressor(cfg.Compression, cfg.CompressLevel),
	}
}

// Config returns the engine configuration.
func (e *Engine) Config() Config { return e.cfg }

// Compressor returns the compression handler.
func (e *Engine) Compressor() *Compressor { return e.compressor }

// WalkDir returns the file listing for a directory tree.
// Uses plain Walk (no SHA256) to avoid reading all file data on listing,
// which would be fatal for large (40GB+) directories.
func (e *Engine) WalkDir(root string) ([]*proto.FileInfo, error) {
	files, err := filesvc.Walk(root)
	if err != nil {
		return nil, err
	}
	log.Printf("walk: %d files (%d dirs), %s total",
		filesvc.FileCount(files),
		len(files)-filesvc.FileCount(files),
		formatBytes(filesvc.TotalSize(files)))
	return files, nil
}

// TransferResult holds the outcome of a transfer.
type TransferResult struct {
	Path      string
	Size      int64
	Duration  time.Duration
	BytesSent int64
	Error     error
}

// SendFiles transmits multiple files to a client via the TCP encoder.
// Uses a worker pool for parallel transfers.
func (e *Engine) SendFiles(ctx context.Context, enc *proto.Encoder, rootDir string, files []*proto.FileInfo) []TransferResult {
	results := make([]TransferResult, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.cfg.Concurrency)

	sender := NewChunkSender(rootDir, e.cfg.ChunkSize, e.cfg.LargeThreshold)

	for i, fi := range files {
		if fi.IsDir {
			results[i] = TransferResult{Path: fi.Path, Error: nil}
			continue
		}

		wg.Add(1)
		go func(idx int, f *proto.FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			start := time.Now()
			err := sender.SendFile(enc, f)
			results[idx] = TransferResult{
				Path:      f.Path,
				Size:      f.Size,
				Duration:  time.Since(start),
				BytesSent: f.Size,
				Error:     err,
			}
			if err != nil {
				log.Printf("ERROR sending %s: %v", f.Path, err)
			}
		}(i, fi)
	}

	wg.Wait()

	// Count statistics
	var totalBytes int64
	var failed int
	for _, r := range results {
		if r.Error != nil {
			failed++
		} else if !r.PathIsDir(files) {
			totalBytes += r.BytesSent
		}
	}

	if failed > 0 {
		log.Printf("Transfer complete: %d/%d files failed, %s sent",
			failed, len(files), formatBytes(totalBytes))
	} else {
		log.Printf("Transfer complete: %d files, %s sent",
			len(files), formatBytes(totalBytes))
	}

	return results
}

// PathIsDir is a helper to check if a TransferResult was for a directory.
func (r *TransferResult) PathIsDir(files []*proto.FileInfo) bool {
	for _, f := range files {
		if f.Path == r.Path {
			return f.IsDir
		}
	}
	return false
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n2 := n / unit; n2 >= unit; n2 /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

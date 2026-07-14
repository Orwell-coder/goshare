package client

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Progress tracks download progress for multiple files.
type Progress struct {
	mu        sync.Mutex
	files     map[string]*fileProgress
	start     time.Time
	isTerm    bool
	termWidth int
}

type fileProgress struct {
	path     string
	size     int64
	received int64
	done     bool
	err      error
}

// NewProgress creates a progress tracker.
func NewProgress() *Progress {
	width, isTerm := getTerminalWidth()
	if width < 40 {
		width = 80
	}
	return &Progress{
		files:     make(map[string]*fileProgress),
		start:     time.Now(),
		isTerm:    isTerm,
		termWidth: width,
	}
}

// AddFile registers a file for progress tracking.
func (p *Progress) AddFile(path string, size int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.files[path] = &fileProgress{
		path: path,
		size: size,
	}
}

// Update updates the received bytes for a file.
func (p *Progress) Update(path string, received int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fp, ok := p.files[path]; ok {
		fp.received = received
	}
}

// Done marks a file as complete.
func (p *Progress) Done(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fp, ok := p.files[path]; ok {
		fp.done = true
		fp.received = fp.size
	}
}

// Error marks a file with an error.
func (p *Progress) Error(path string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fp, ok := p.files[path]; ok {
		fp.err = err
		fp.done = true
	}
}

// Render prints the current progress to stdout.
func (p *Progress) Render() {
	p.mu.Lock()
	defer p.mu.Unlock()

	var totalSize, totalRecv int64
	var doneCount, errCount int

	for _, fp := range p.files {
		totalSize += fp.size
		totalRecv += fp.received
		if fp.done {
			if fp.err != nil {
				errCount++
			} else {
				doneCount++
			}
		}
	}

	elapsed := time.Since(p.start)
	speed := float64(0)
	if elapsed.Seconds() > 0 {
		speed = float64(totalRecv) / elapsed.Seconds()
	}

	if !p.isTerm {
		// Non-terminal: carriage return line
		fmt.Printf("\r[%d/%d] %s / %s  %s/s",
			doneCount, len(p.files),
			formatBytes(totalRecv), formatBytes(totalSize),
			formatBytes(int64(speed)))
		return
	}

	// Terminal: animated progress bar
	barWidth := p.termWidth - 45
	if barWidth < 10 {
		barWidth = 10
	}

	ratio := float64(0)
	if totalSize > 0 {
		ratio = float64(totalRecv) / float64(totalSize)
	}

	filled := int(ratio * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	line := fmt.Sprintf("\r[%d/%d] %s %5.1f%%  %s / %s  %s/s  %s",
		doneCount, len(p.files), bar, ratio*100,
		formatBytes(totalRecv), formatBytes(totalSize),
		formatBytes(int64(speed)),
		elapsed.Truncate(time.Second).String())

	// Pad to clear previous line content
	if len(line) < p.termWidth {
		line += strings.Repeat(" ", p.termWidth-len(line))
	}
	fmt.Print(line)
}

// Summary prints the final download summary.
func (p *Progress) Summary() {
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.start)
	var totalSize, totalRecv int64
	var doneCount, errCount int

	for _, fp := range p.files {
		totalSize += fp.size
		totalRecv += fp.received
		if fp.done {
			if fp.err != nil {
				errCount++
			} else {
				doneCount++
			}
		}
	}

	speed := float64(0)
	if elapsed.Seconds() > 0 {
		speed = float64(totalRecv) / elapsed.Seconds()
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  完成！总计 %d 文件", doneCount)
	if errCount > 0 {
		fmt.Printf(", %d 失败", errCount)
	}
	fmt.Println()
	fmt.Printf("  数据量: %s\n", formatBytes(totalSize))
	fmt.Printf("  耗时:   %s\n", elapsed.Truncate(time.Second))
	fmt.Printf("  平均速度: %s/s\n", formatBytes(int64(speed)))
	fmt.Println(strings.Repeat("=", 60))
}

func formatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	switch {
	case n >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

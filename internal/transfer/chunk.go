package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/Orwell-coder/goshare/internal/proto"
)

// DefaultChunkSize is the default size of data chunks sent over TCP (4MB).
const DefaultChunkSize = 4 * 1024 * 1024

// ChunkSender streams files to a proto.Encoder in fixed-size chunks.
// All files, regardless of size, are streamed — no full-file reads into memory.
type ChunkSender struct {
	chunkSize int
	rootDir   string
}

// Buffer pool to reuse chunk buffers and avoid GC pressure during large transfers.
var chunkBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, DefaultChunkSize)
	},
}

// NewChunkSender creates a ChunkSender.
func NewChunkSender(rootDir string, chunkSize int, _ int64) *ChunkSender {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &ChunkSender{
		chunkSize: chunkSize,
		rootDir:   rootDir,
	}
}

// SendFile streams a single file to the encoder in chunks.
// Every file is streamed, never fully loaded into memory.
func (s *ChunkSender) SendFile(enc *proto.Encoder, fi *proto.FileInfo) (err error) {
	absPath := filepath.Join(s.rootDir, filepath.FromSlash(fi.Path))

	f, openErr := os.Open(absPath)
	if openErr != nil {
		return fmt.Errorf("open %s: %w", fi.Path, openErr)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Empty file: send a single empty chunk so the client creates the file.
	if fi.Size == 0 {
		return enc.WriteChunk(fi.Path, 0, nil, true)
	}

	buf := chunkBufPool.Get().([]byte)
	defer chunkBufPool.Put(buf)

	// Ensure buffer is at least chunkSize
	if cap(buf) < s.chunkSize {
		buf = make([]byte, s.chunkSize)
	}

	var offset int64
	for offset < fi.Size {
		toRead := int64(len(buf))
		if remaining := fi.Size - offset; remaining < toRead {
			toRead = remaining
		}
		if int(toRead) > cap(buf) {
			toRead = int64(cap(buf))
		}

		n, readErr := f.ReadAt(buf[:toRead], offset)
		if readErr != nil && readErr != io.EOF {
			return fmt.Errorf("read %s at %d: %w", fi.Path, offset, readErr)
		}
		if n == 0 {
			break
		}

		isLast := offset+int64(n) >= fi.Size
		if writeErr := enc.WriteChunk(fi.Path, offset, buf[:n], isLast); writeErr != nil {
			return fmt.Errorf("write chunk %s at %d: %w", fi.Path, offset, writeErr)
		}
		offset += int64(n)
	}
	return nil
}

// SendBatch transmits multiple files in sequence.
func (s *ChunkSender) SendBatch(enc *proto.Encoder, files []*proto.FileInfo) error {
	for _, fi := range files {
		if fi.IsDir {
			continue
		}
		if err := s.SendFile(enc, fi); err != nil {
			return err
		}
	}
	return nil
}

// WriteFileChunk writes received chunk data to a local file.
// Creates parent directories as needed.
func WriteFileChunk(baseDir string, chunk *proto.DataChunk) error {
	localPath := filepath.Join(baseDir, filepath.FromSlash(chunk.Path))

	if chunk.Offset == 0 {
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return err
		}
	}

	flag := os.O_CREATE | os.O_WRONLY
	if chunk.Offset == 0 {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(localPath, flag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteAt(chunk.Data, chunk.Offset)
	return err
}

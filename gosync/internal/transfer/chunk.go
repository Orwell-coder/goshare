package transfer

import (
	"os"
	"path/filepath"

	"gosync/internal/proto"
)

// ChunkSize is the default size of data chunks sent over TCP.
// 4MB balances throughput vs memory per connection.
const DefaultChunkSize = 4 * 1024 * 1024 // 4MB

// LargeFileThreshold: files above this size use chunked sendfile.
const DefaultLargeFileThreshold = 16 * 1024 * 1024 // 16MB

// ChunkSender manages chunked file transmission over a proto.Encoder.
type ChunkSender struct {
	chunkSize       int
	largeThreshold  int64
	rootDir         string
}

// NewChunkSender creates a ChunkSender.
func NewChunkSender(rootDir string, chunkSize int, largeSizeThreshold int64) *ChunkSender {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if largeSizeThreshold <= 0 {
		largeSizeThreshold = DefaultLargeFileThreshold
	}
	return &ChunkSender{
		chunkSize:      chunkSize,
		largeThreshold: largeSizeThreshold,
		rootDir:        rootDir,
	}
}

// SendFile transmits a single file in chunks. For files smaller than the large
// threshold, it reads the entire file and sends as one chunk. For large files,
// it streams in chunkSize blocks from disk.
func (s *ChunkSender) SendFile(enc *proto.Encoder, fi *proto.FileInfo) error {
	absPath := filepath.Join(s.rootDir, filepath.FromSlash(fi.Path))

	f, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if fi.Size < s.largeThreshold {
		// Small file: read all at once
		data := make([]byte, fi.Size)
		if _, err := f.Read(data); err != nil {
			return err
		}
		return enc.WriteChunk(fi.Path, 0, data, true)
	}

	// Large file: stream in chunks
	buf := make([]byte, s.chunkSize)
	var offset int64
	for offset < fi.Size {
		toRead := int64(len(buf))
		if remaining := fi.Size - offset; remaining < toRead {
			toRead = remaining
			buf = buf[:toRead]
		}
		n, err := f.ReadAt(buf[:toRead], offset)
		if err != nil && err != os.ErrClosed {
			return err
		}
		if n == 0 {
			break
		}
		isLast := offset+int64(n) >= fi.Size
		if err := enc.WriteChunk(fi.Path, offset, buf[:n], isLast); err != nil {
			return err
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

// WriteFile writes received chunk data to a local file.
// Creates parent directories as needed.
func WriteFileChunk(baseDir string, chunk *proto.DataChunk) error {
	localPath := filepath.Join(baseDir, filepath.FromSlash(chunk.Path))

	// Create parent directories on first chunk
	if chunk.Offset == 0 {
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return err
		}
	}

	// Open file for writing at the correct offset
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

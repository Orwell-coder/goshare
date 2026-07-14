package filesvc

import (
	"crypto/sha256"
	"io"
	"os"

	"github.com/zhengxu/goshare/internal/proto"
)

// FileInfo extends proto.FileInfo with local filesystem helpers.
type FileInfo = proto.FileInfo

// Checksum computes SHA-256 hash of a file. Returns nil, nil for directories.
func Checksum(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// Exists checks if a local file exists and has the same size and modification time.
func Exists(localPath string, fi *FileInfo) bool {
	info, err := os.Stat(localPath)
	if err != nil {
		return false
	}
	if info.IsDir() != fi.IsDir {
		return false
	}
	if !fi.IsDir && info.Size() != fi.Size {
		return false
	}
	// ModTime comparison with 1-second tolerance (FAT/exFAT compatibility)
	diff := info.ModTime().UnixNano() - fi.ModTime
	if diff < 0 {
		diff = -diff
	}
	return diff < 1_000_000_000 // 1 second tolerance
}

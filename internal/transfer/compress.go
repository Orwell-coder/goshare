package transfer

import (
	"bytes"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// Compressor wraps zstd compression with encoder/decoder pools.
type Compressor struct {
	enabled   bool
	level     int
	encPool   sync.Pool
	decPool   sync.Pool
}

// NewCompressor creates a compressor. If enabled is false, compression is skipped.
func NewCompressor(enabled bool, level int) *Compressor {
	if !enabled {
		return &Compressor{enabled: false}
	}
	if level < 1 {
		level = 3
	}
	if level > 22 {
		level = 22
	}
	c := &Compressor{enabled: true, level: level}
	c.encPool.New = func() interface{} {
		enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
		return enc
	}
	c.decPool.New = func() interface{} {
		dec, _ := zstd.NewReader(nil)
		return dec
	}
	return c
}

// Compress compresses data. Returns original data if compression is disabled
// or if compressed size is not smaller.
func (c *Compressor) Compress(data []byte) ([]byte, bool) {
	if !c.enabled || len(data) < 256 {
		return data, false
	}

	// Skip already-compressed formats
	if isCompressedFormat(data) {
		return data, false
	}

	enc := c.encPool.Get().(*zstd.Encoder)
	defer c.encPool.Put(enc)

	compressed := enc.EncodeAll(data, make([]byte, 0, len(data)/2))
	if len(compressed) >= len(data) {
		return data, false // compression didn't help
	}
	return compressed, true
}

// Decompress decompresses data that was compressed with Compress.
func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	dec := c.decPool.Get().(*zstd.Decoder)
	defer c.decPool.Put(dec)
	return dec.DecodeAll(data, nil)
}

// isCompressedFormat detects common already-compressed file formats by magic bytes.
func isCompressedFormat(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// ZIP, RAR, 7z
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return true
	}
	if bytes.HasPrefix(data, []byte("Rar!\x1a\x07")) {
		return true
	}
	if bytes.HasPrefix(data, []byte("7z\xbc\xaf\x27\x1c")) {
		return true
	}
	// gzip
	if bytes.HasPrefix(data, []byte("\x1f\x8b")) {
		return true
	}
	// JPEG
	if bytes.HasPrefix(data, []byte("\xff\xd8\xff")) {
		return true
	}
	// PNG
	if bytes.HasPrefix(data, []byte("\x89PNG")) {
		return true
	}
	// GIF
	if bytes.HasPrefix(data, []byte("GIF8")) {
		return true
	}
	// MP4 / common video
	if bytes.HasPrefix(data[4:], []byte("ftyp")) && data[0] == 0x00 {
		return true
	}
	// WebP
	if bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 8 && bytes.HasPrefix(data[8:], []byte("WEBP")) {
		return true
	}
	// zstd (already compressed by us)
	if bytes.HasPrefix(data, []byte("\x28\xb5\x2f\xfd")) {
		return true
	}
	return false
}

// CompressedWriter wraps an io.Writer with zstd compression.
type CompressedWriter struct {
	enc *zstd.Encoder
	w   io.Writer
}

func (c *Compressor) NewCompressedWriter(w io.Writer) io.WriteCloser {
	enc, _ := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.EncoderLevel(c.level)))
	return enc
}

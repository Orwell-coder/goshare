package transfer

import (
	"io"
	"net"
	"os"
	"sync"
)

// sendBufPool provides large buffers for io.CopyBuffer to approximate zero-copy.
// Using a 1MB buffer is sufficient for saturating gigabit LAN.
var sendBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1024*1024) // 1MB
	},
}

// sendFile transmits a file over a TCP connection with a large buffer.
// On Windows, TransmitFile API is available via mswsock.dll but requires
// additional syscall setup. Using a large-buffer io.CopyBuffer achieves
// comparable LAN throughput (typically 100+ MB/s on gigabit).
// For Linux, replace with syscall.Sendfile for true zero-copy.
func sendFile(conn net.Conn, f *os.File, offset, size int64) (int64, error) {
	buf := sendBufPool.Get().([]byte)
	defer sendBufPool.Put(buf)

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}

	// Wrap in LimitReader so we send exactly 'size' bytes from the offset.
	lr := io.LimitReader(f, size)
	return io.CopyBuffer(conn, lr, buf)
}

// sendFileAll sends the entire file from the beginning.
func sendFileAll(conn net.Conn, f *os.File) (int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return sendFile(conn, f, 0, fi.Size())
}

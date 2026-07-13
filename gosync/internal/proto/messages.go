package proto

// MessageType identifies the type of message in the TCP protocol.
type MessageType byte

const (
	TypeListRequest    MessageType = 1
	TypeListResponse   MessageType = 2
	TypeDownloadReq    MessageType = 3
	TypeDataChunk      MessageType = 4
	TypeBatchDone      MessageType = 5
	TypeError          MessageType = 6
)

// FileInfo describes a single file or directory in the remote tree.
type FileInfo struct {
	Path    string
	Size    int64
	ModTime int64 // unix nano
	SHA256  []byte // 32 bytes, nil for directories
	IsDir   bool
}

// ListRequest asks the server for the file tree under a given path.
type ListRequest struct {
	Path string
}

// ListResponse returns the file tree.
type ListResponse struct {
	Files   []*FileInfo
	RootDir string
}

// DownloadReq requests a batch of files to download.
type DownloadReq struct {
	Files []string // relative paths to download
}

// DataChunk carries a chunk of file data.
// Sent as raw binary (not gob) for zero-overhead bulk transfer.
type DataChunk struct {
	Path   string
	Offset int64
	Data   []byte
	IsLast bool // true if this is the last chunk for this file
}

// BatchDone signals that all requested files have been fully sent.
type BatchDone struct {
	Path string
}

// ErrorMessage carries an error from the server to the client.
type ErrorMessage struct {
	Code    int32
	Message string
}

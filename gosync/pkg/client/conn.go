package client

import (
	"fmt"
	"net"
	"time"

	"gosync/internal/proto"
)

// Conn wraps a TCP connection to the GoSync server with message encoding.
type Conn struct {
	conn net.Conn
	enc  *proto.Encoder
	dec  *proto.Decoder
}

// Connect establishes a TCP connection to the server.
func Connect(host string, port int) (*Conn, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接到 %s 失败: %w", addr, err)
	}

	// Set TCP options for high throughput
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetReadBuffer(1024 * 1024)  // 1MB
		tcpConn.SetWriteBuffer(1024 * 1024) // 1MB
	}

	return &Conn{
		conn: conn,
		enc:  proto.NewEncoder(conn),
		dec:  proto.NewDecoder(conn),
	}, nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// List requests the file tree from the server for a given path.
// maxDepth controls recursion: 0 = unlimited, 1 = direct children only.
func (c *Conn) List(path string, maxDepth int) (*proto.ListResponse, error) {
	if err := c.enc.Encode(&proto.ListRequest{Path: path, MaxDepth: maxDepth}); err != nil {
		return nil, fmt.Errorf("发送列表请求失败: %w", err)
	}

	mt, msg, err := c.dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("读取列表响应失败: %w", err)
	}

	switch mt {
	case proto.TypeListResponse:
		return msg.(*proto.ListResponse), nil
	case proto.TypeError:
		errMsg := msg.(*proto.ErrorMessage)
		return nil, fmt.Errorf("服务器错误: %s", errMsg.Message)
	default:
		return nil, fmt.Errorf("意外的响应类型: %d", mt)
	}
}

// Download requests a batch of files from the server.
// The callback is called for each received DataChunk.
func (c *Conn) Download(files []string, onChunk func(*proto.DataChunk) error) error {
	if err := c.enc.Encode(&proto.DownloadReq{Files: files}); err != nil {
		return fmt.Errorf("发送下载请求失败: %w", err)
	}

	remaining := len(files)
	for remaining > 0 {
		mt, msg, err := c.dec.Decode()
		if err != nil {
			return fmt.Errorf("读取下载数据失败: %w", err)
		}

		switch mt {
		case proto.TypeDataChunk:
			chunk := msg.(*proto.DataChunk)
			if err := onChunk(chunk); err != nil {
				return fmt.Errorf("处理数据块失败: %w", err)
			}
		case proto.TypeBatchDone:
			remaining--
		case proto.TypeError:
			errMsg := msg.(*proto.ErrorMessage)
			return fmt.Errorf("服务器错误: %s", errMsg.Message)
		default:
			return fmt.Errorf("意外的消息类型: %d", mt)
		}
	}

	return nil
}

// RemoteAddr returns the server address.
func (c *Conn) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

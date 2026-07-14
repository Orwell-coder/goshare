package proto

import (
	"bufio"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

func init() {
	gob.Register(&ListRequest{})
	gob.Register(&ListResponse{})
	gob.Register(&DownloadReq{})
	gob.Register(&BatchDone{})
	gob.Register(&ErrorMessage{})
}

// Encoder writes typed messages to a TCP connection.
// Control messages use gob encoding; data chunks use raw binary for zero-overhead.
type Encoder struct {
	w   *bufio.Writer
	enc *gob.Encoder
}

// Decoder reads typed messages from a TCP connection.
type Decoder struct {
	r   *bufio.Reader
	dec *gob.Decoder
}

// NewCodec creates an Encoder/Decoder pair for a connection.
func NewEncoder(conn net.Conn) *Encoder {
	w := bufio.NewWriterSize(conn, 8*1024*1024) // 8MB: fits two 4MB chunks + headers without intermediate flush
	return &Encoder{
		w:   w,
		enc: gob.NewEncoder(w),
	}
}

func NewDecoder(conn net.Conn) *Decoder {
	r := bufio.NewReaderSize(conn, 1024*1024) // 1MB read buffer
	return &Decoder{
		r:   r,
		dec: gob.NewDecoder(r),
	}
}

// Encode writes a control message (type byte + gob payload).
func (e *Encoder) Encode(msg interface{}) error {
	var mt MessageType
	switch msg.(type) {
	case *ListRequest:
		mt = TypeListRequest
	case *ListResponse:
		mt = TypeListResponse
	case *DownloadReq:
		mt = TypeDownloadReq
	case *BatchDone:
		mt = TypeBatchDone
	case *ErrorMessage:
		mt = TypeError
	default:
		return fmt.Errorf("unsupported message type: %T", msg)
	}
	if err := e.w.WriteByte(byte(mt)); err != nil {
		return err
	}
	if err := e.enc.Encode(msg); err != nil {
		return err
	}
	return e.w.Flush()
}

// WriteChunk sends a file data chunk using raw binary encoding.
// Format: [type:1B][pathLen:2B][path:N][offset:8B][dataLen:4B][data:N][isLast:1B]
// Does NOT flush — BatchDone.Encode() handles that, reducing syscalls per chunk from ~5 to ~0.
func (e *Encoder) WriteChunk(path string, offset int64, data []byte, isLast bool) error {
	if err := e.w.WriteByte(byte(TypeDataChunk)); err != nil {
		return err
	}

	// Hand-rolled binary encoding avoids encoding/binary reflection overhead.
	var hdr [14]byte
	binary.BigEndian.PutUint16(hdr[0:2], uint16(len(path)))
	binary.BigEndian.PutUint64(hdr[2:10], uint64(offset))
	binary.BigEndian.PutUint32(hdr[10:14], uint32(len(data)))

	if _, err := e.w.Write(hdr[0:2]); err != nil {
		return err
	}
	if _, err := e.w.WriteString(path); err != nil {
		return err
	}
	if _, err := e.w.Write(hdr[2:14]); err != nil {
		return err
	}
	if _, err := e.w.Write(data); err != nil {
		return err
	}
	if isLast {
		return e.w.WriteByte(1)
	}
	return e.w.WriteByte(0)
}

// Decode reads and returns the next message from the stream.
func (d *Decoder) Decode() (MessageType, interface{}, error) {
	mt, err := d.r.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	switch MessageType(mt) {
	case TypeListRequest:
		var msg ListRequest
		if err := d.dec.Decode(&msg); err != nil {
			return 0, nil, err
		}
		return TypeListRequest, &msg, nil
	case TypeListResponse:
		var msg ListResponse
		if err := d.dec.Decode(&msg); err != nil {
			return 0, nil, err
		}
		return TypeListResponse, &msg, nil
	case TypeDownloadReq:
		var msg DownloadReq
		if err := d.dec.Decode(&msg); err != nil {
			return 0, nil, err
		}
		return TypeDownloadReq, &msg, nil
	case TypeBatchDone:
		var msg BatchDone
		if err := d.dec.Decode(&msg); err != nil {
			return 0, nil, err
		}
		return TypeBatchDone, &msg, nil
	case TypeError:
		var msg ErrorMessage
		if err := d.dec.Decode(&msg); err != nil {
			return 0, nil, err
		}
		return TypeError, &msg, nil
	case TypeDataChunk:
		chunk, err := d.decodeChunk()
		return TypeDataChunk, chunk, err
	default:
		return 0, nil, fmt.Errorf("unknown message type: 0x%02x", mt)
	}
}

func (d *Decoder) decodeChunk() (*DataChunk, error) {
	var pathLen uint16
	if err := binary.Read(d.r, binary.BigEndian, &pathLen); err != nil {
		return nil, fmt.Errorf("chunk pathLen: %w", err)
	}
	pathBytes := make([]byte, pathLen)
	if _, err := io.ReadFull(d.r, pathBytes); err != nil {
		return nil, fmt.Errorf("chunk path: %w", err)
	}
	var offset int64
	if err := binary.Read(d.r, binary.BigEndian, &offset); err != nil {
		return nil, fmt.Errorf("chunk offset: %w", err)
	}
	var dataLen uint32
	if err := binary.Read(d.r, binary.BigEndian, &dataLen); err != nil {
		return nil, fmt.Errorf("chunk dataLen: %w", err)
	}
	data := make([]byte, dataLen)
	if _, err := io.ReadFull(d.r, data); err != nil {
		return nil, fmt.Errorf("chunk data: %w", err)
	}
	isLastByte, err := d.r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("chunk isLast: %w", err)
	}
	return &DataChunk{
		Path:   string(pathBytes),
		Offset: offset,
		Data:   data,
		IsLast: isLastByte == 1,
	}, nil
}

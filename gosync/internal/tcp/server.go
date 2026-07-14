package tcp

import (
	"context"
	"log"
	"net"
	"runtime/debug"
	"time"

	"github.com/zhengxu/gosync/internal/proto"
	"github.com/zhengxu/gosync/internal/transfer"
)

// Server handles TCP client connections for high-speed file transfer.
type Server struct {
	cfg      transfer.Config
	engine   *transfer.Engine
	rootDirs []string
	ln       net.Listener
}

// NewServer creates a TCP server.
func NewServer(cfg transfer.Config, engine *transfer.Engine, rootDirs []string) *Server {
	return &Server{
		cfg:      cfg,
		engine:   engine,
		rootDirs: rootDirs,
	}
}

// ListenAndServe starts the TCP server and blocks until the context is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	lc := net.ListenConfig{
		KeepAlive: 30 * time.Second,
	}
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("[tcp] listening on %s", addr)

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("[tcp] accept error: %v", err)
				continue
			}
		}

		// Set TCP options for high throughput
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetNoDelay(true) // disable Nagle for low latency
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
			// Larger buffers for LAN throughput
			tcpConn.SetReadBuffer(1024 * 1024)  // 1MB
			tcpConn.SetWriteBuffer(1024 * 1024) // 1MB
		}

		log.Printf("[tcp] new connection from %s", conn.RemoteAddr())
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[tcp] PANIC in session from %s: %v\n%s",
				conn.RemoteAddr(), r, string(debug.Stack()))
		}
	}()

	enc := proto.NewEncoder(conn)
	dec := proto.NewDecoder(conn)

	session := NewSession(enc, dec, s.engine, s.rootDirs)
	if err := session.Handle(ctx); err != nil {
		if !isEOF(err) {
			log.Printf("[tcp] session ended: %v", err)
		}
	}
}

// Addr returns the listening address.
func (s *Server) Addr() net.Addr {
	if s.ln != nil {
		return s.ln.Addr()
	}
	return nil
}

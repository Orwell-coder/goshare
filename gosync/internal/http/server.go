package http

import (
	"context"
	"log"
	"net"
	"net/http"
)

// Server serves the HTTP interface for browser access.
type Server struct {
	srv      *http.Server
	rootDirs []string
}

// NewServer creates an HTTP server.
func NewServer(rootDirs []string) *Server {
	s := &Server{rootDirs: rootDirs}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleBrowse(rootDirs))
	mux.HandleFunc("/browse/", handleBrowse(rootDirs))
	mux.HandleFunc("/download", handleDownload(rootDirs))

	s.srv = &http.Server{
		Handler:      mux,
		ConnContext:  saveConnInContext,
		ReadTimeout:  0, // streaming downloads have no timeout
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	return s
}

// ListenAndServe starts the HTTP server and blocks until the context is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("[http] listening on http://%s", addr)

	go func() {
		<-ctx.Done()
		s.srv.Close()
	}()

	err = s.srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// saveConnInContext is a no-op adapter for compatibility.
func saveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return ctx
}

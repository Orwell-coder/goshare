package http

import (
	"context"
	"log"
	"net"
	"net/http"
	"runtime/debug"
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
		Handler:      recoveryMiddleware(mux),
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

// recoveryMiddleware catches panics in HTTP handlers and logs them.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[http] PANIC in %s %s: %v\n%s",
					r.Method, r.URL.Path, rec, string(debug.Stack()))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

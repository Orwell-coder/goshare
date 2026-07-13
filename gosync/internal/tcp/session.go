package tcp

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gosync/internal/proto"
	"gosync/internal/transfer"
)

// Session handles a single client TCP connection.
type Session struct {
	enc      *proto.Encoder
	dec      *proto.Decoder
	engine   *transfer.Engine
	rootDirs []string
}

// NewSession creates a session for a client connection.
func NewSession(enc *proto.Encoder, dec *proto.Decoder, engine *transfer.Engine, rootDirs []string) *Session {
	return &Session{
		enc:      enc,
		dec:      dec,
		engine:   engine,
		rootDirs: rootDirs,
	}
}

// Handle processes incoming messages until the connection closes or context is cancelled.
func (s *Session) Handle(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		mt, msg, err := s.dec.Decode()
		if err != nil {
			if !isEOF(err) {
				log.Printf("[tcp] decode error: %v", err)
			}
			return err
		}

		switch mt {
		case proto.TypeListRequest:
			if err := s.handleList(msg.(*proto.ListRequest)); err != nil {
				log.Printf("[tcp] list error: %v", err)
				return err
			}
		case proto.TypeDownloadReq:
			if err := s.handleDownload(msg.(*proto.DownloadReq)); err != nil {
				log.Printf("[tcp] download error: %v", err)
				return err
			}
		default:
			log.Printf("[tcp] unexpected message type: %d", mt)
		}
	}
}

func (s *Session) handleList(req *proto.ListRequest) error {
	root := s.resolveRoot(req.Path)
	if root == "" {
		return s.enc.Encode(&proto.ErrorMessage{
			Code:    404,
			Message: "path not accessible: " + req.Path,
		})
	}

	files, err := s.engine.WalkDir(root)
	if err != nil {
		return s.enc.Encode(&proto.ErrorMessage{
			Code:    500,
			Message: "walk failed: " + err.Error(),
		})
	}

	return s.enc.Encode(&proto.ListResponse{
		Files:   files,
		RootDir: filepath.ToSlash(req.Path),
	})
}

func (s *Session) handleDownload(req *proto.DownloadReq) error {
	for _, relPath := range req.Files {
		absPath := s.resolveFilePath(relPath)
		if absPath == "" {
			s.enc.Encode(&proto.ErrorMessage{
				Code:    404,
				Message: "file not found: " + relPath,
			})
			continue
		}

		fi, err := os.Stat(absPath)
		if err != nil {
			s.enc.Encode(&proto.ErrorMessage{
				Code:    404,
				Message: "stat failed: " + relPath,
			})
			continue
		}

		if fi.IsDir() {
			continue
		}

		rootDir := s.findRootDir(absPath)

		sender := transfer.NewChunkSender(
			rootDir,
			s.engine.Config().ChunkSize,
			s.engine.Config().LargeThreshold,
		)

		fileInfo := &proto.FileInfo{
			Path:  relPath,
			Size:  fi.Size(),
			IsDir: false,
		}

		if err := sender.SendFile(s.enc, fileInfo); err != nil {
			log.Printf("[tcp] send error for %s: %v", relPath, err)
			s.enc.Encode(&proto.ErrorMessage{
				Code:    500,
				Message: "send failed: " + relPath + ": " + err.Error(),
			})
			continue
		}

		if err := s.enc.Encode(&proto.BatchDone{Path: relPath}); err != nil {
			return err
		}
	}

	return nil
}

// resolveRoot resolves a client-requested path (e.g. "/testdata" or "/data/movies")
// to an absolute directory path within the configured root directories.
func (s *Session) resolveRoot(requestPath string) string {
	relPath := filepath.FromSlash(strings.Trim(requestPath, "/"))
	if relPath == "" || relPath == "." {
		return s.rootDirs[0]
	}

	for _, root := range s.rootDirs {
		candidate := filepath.Join(root, relPath)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		if filepath.Base(root) == relPath {
			return root
		}
	}

	return ""
}

// resolveFilePath resolves a relative file path against all root directories.
func (s *Session) resolveFilePath(relPath string) string {
	cleanPath := filepath.FromSlash(strings.Trim(relPath, "/"))

	for _, root := range s.rootDirs {
		candidate := filepath.Join(root, cleanPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// findRootDir returns the root directory that contains the given absolute path.
func (s *Session) findRootDir(absPath string) string {
	for _, root := range s.rootDirs {
		cleanRoot := filepath.Clean(root)
		cleanAbs := filepath.Clean(absPath)
		if strings.HasPrefix(cleanAbs, cleanRoot+string(filepath.Separator)) || cleanAbs == cleanRoot {
			return root
		}
	}
	return filepath.Dir(absPath)
}

// isEOF returns true if the error indicates a normal connection close.
func isEOF(err error) bool {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	// Also catch wrapped EOF
	s := err.Error()
	return s == "EOF" || strings.Contains(s, "EOF")
}

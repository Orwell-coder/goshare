package http

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleDownload streams a directory as a zip file to the HTTP response.
// URL format: /download?path=/<rootName>/<subpath...>
func handleDownload(rootDirs []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pathParam := r.URL.Query().Get("path")
		if pathParam == "" {
			http.Error(w, "缺少 path 参数", http.StatusBadRequest)
			return
		}

		relPath := strings.Trim(pathParam, "/")

		// Path format: <rootName>/<subpath...>
		parts := strings.SplitN(relPath, "/", 2)
		rootName := parts[0]
		subPath := ""
		if len(parts) > 1 {
			subPath = parts[1]
		}

		// Find the root by name and resolve subpath
		var targetDir string
		for _, root := range rootDirs {
			if filepath.Base(root) != rootName {
				continue
			}
			if subPath == "" {
				targetDir = root
			} else {
				candidate := filepath.Join(root, filepath.FromSlash(subPath))
				if info, err := os.Stat(candidate); err == nil {
					if info.IsDir() {
						targetDir = candidate
					} else {
						serveFile(w, r, candidate)
						return
					}
				}
			}
			break
		}

		if targetDir == "" {
			http.NotFound(w, r)
			return
		}

		// Stream the directory as a zip
		dirName := filepath.Base(targetDir)
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s.zip"`, dirName))

		zw := zip.NewWriter(w)
		defer zw.Close()

		prefix := targetDir + string(filepath.Separator)

		err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			zipPath := strings.TrimPrefix(path, prefix)
			zipPath = filepath.ToSlash(zipPath)

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(filepath.Join(dirName, zipPath))
			header.Method = zip.Deflate

			writer, err := zw.CreateHeader(header)
			if err != nil {
				return err
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			written, err := io.Copy(writer, f)
			if err != nil {
				return err
			}
			log.Printf("[http] zipped %s (%d bytes)", zipPath, written)
			return nil
		})

		if err != nil {
			log.Printf("[http] zip stream error: %v", err)
		}
	}
}

// serveFile serves a single file for download.
func serveFile(w http.ResponseWriter, r *http.Request, filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, _ := f.Stat()
	name := filepath.Base(filePath)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, name))
	if info != nil {
		w.Header().Set("Content-Length", itoa(info.Size()))
	}

	io.Copy(w, f)
}

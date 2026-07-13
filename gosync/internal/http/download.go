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
func handleDownload(rootDirs []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pathParam := r.URL.Query().Get("path")
		if pathParam == "" {
			http.Error(w, "缺少 path 参数", http.StatusBadRequest)
			return
		}

		relPath := strings.TrimPrefix(pathParam, "/")
		cleanPath := filepath.FromSlash(relPath)

		// Resolve the target directory across all root dirs
		var targetDir string
		for _, root := range rootDirs {
			candidate := filepath.Join(root, cleanPath)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				targetDir = candidate
				break
			}
			// Try as a root name
			if filepath.Base(root) == cleanPath {
				targetDir = root
				break
			}
		}

		if targetDir == "" {
			// Try as a file
			for _, root := range rootDirs {
				candidate := filepath.Join(root, cleanPath)
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					// Single file download
					serveFile(w, r, candidate)
					return
				}
			}
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

			// Relative path within the zip
			zipPath := strings.TrimPrefix(path, prefix)
			zipPath = filepath.ToSlash(zipPath)

			// Create zip header
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

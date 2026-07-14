package filesvc

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/Orwell-coder/goshare/internal/proto"
)

// Walk recursively walks a directory tree and returns all FileInfo entries sorted by path.
// Directory entries come before their children for correct client-side mkdir ordering.
func Walk(root string) ([]*proto.FileInfo, error) {
	var (
		files []*proto.FileInfo
		mu    sync.Mutex
	)

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		fi := &proto.FileInfo{
			Path:    filepath.ToSlash(relPath),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			IsDir:   info.IsDir(),
		}

		mu.Lock()
		files = append(files, fi)
		mu.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// WalkConcurrent walks a directory tree with concurrent checksum computation.
// Returns files with SHA256 populated for regular files.
func WalkConcurrent(root string, concurrency int) ([]*proto.FileInfo, error) {
	files, err := Walk(root)
	if err != nil {
		return nil, err
	}

	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, fi := range files {
		if fi.IsDir {
			continue
		}
		wg.Add(1)
		go func(f *proto.FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			absPath := filepath.Join(root, filepath.FromSlash(f.Path))
			f.SHA256, _ = Checksum(absPath)
		}(fi)
	}

	wg.Wait()
	return files, nil
}

// FileCount returns the number of regular files in the list.
func FileCount(files []*proto.FileInfo) int {
	n := 0
	for _, f := range files {
		if !f.IsDir {
			n++
		}
	}
	return n
}

// WalkShallow reads only the direct children of root (depth 1), no recursion.
// Directories are sorted before files, then alphabetically by name.
func WalkShallow(root string) ([]*proto.FileInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	files := make([]*proto.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fi := &proto.FileInfo{
			Path:    filepath.ToSlash(entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			IsDir:   entry.IsDir(),
		}
		files = append(files, fi)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// TotalSize returns the total size of all regular files.
func TotalSize(files []*proto.FileInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

package http

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var browseTmpl = template.Must(template.New("browse").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>GoSync — {{.CurrentPath}}</title>
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
       background:#1a1a2e; color:#e0e0e0; min-height:100vh; }
.header { background:#16213e; padding:16px 24px; border-bottom:1px solid #0f3460; }
.header h1 { font-size:18px; font-weight:600; color:#e94560; }
.header .path { font-size:13px; color:#a0a0b0; margin-top:4px; }
.toolbar { padding:12px 24px; background:#16213e; display:flex; gap:12px; align-items:center; }
.toolbar a { color:#e94560; text-decoration:none; font-size:14px; }
.toolbar a:hover { text-decoration:underline; }
table { width:100%; border-collapse:collapse; }
th { text-align:left; padding:10px 24px; font-size:12px; text-transform:uppercase;
     color:#8892b0; border-bottom:1px solid #0f3460; background:#16213e; position:sticky; top:0; }
td { padding:8px 24px; border-bottom:1px solid #0f3460; font-size:14px; }
tr:hover { background:rgba(233,69,96,0.05); }
.name a { color:#64ffda; text-decoration:none; }
.name a:hover { text-decoration:underline; }
.dir a { color:#e94560; font-weight:500; }
.size { color:#8892b0; text-align:right; white-space:nowrap; }
.time { color:#8892b0; white-space:nowrap; }
.dl-btn { background:#e94560; color:#fff; border:none; padding:4px 12px; border-radius:4px;
          cursor:pointer; font-size:12px; text-decoration:none; }
.dl-btn:hover { background:#c73650; }
.empty { text-align:center; padding:48px; color:#8892b0; }
.footer { padding:12px 24px; color:#5a5a7a; font-size:11px; }
</style>
</head>
<body>
<div class="header">
  <h1>GoSync</h1>
  <div class="path">/{{.CurrentPath}}</div>
</div>
<div class="toolbar">
  <a href="/">根目录</a>
  {{if .Parent}}<a href="/browse/{{.Parent}}">⬆ 上级目录</a>{{end}}
  {{if .CurrentPath}}<a class="dl-btn" href="/download?path=/{{.CurrentPath}}">⬇ 下载此文件夹</a>{{end}}
</div>
<table>
<thead><tr><th>名称</th><th style="text-align:right">大小</th><th>修改时间</th><th>操作</th></tr></thead>
<tbody>
{{if .Entries}}
  {{range .Entries}}
  <tr>
    <td class="name {{if .IsDir}}dir{{end}}">
      {{if .IsDir}}
        <a href="/browse/{{$.CurrentPath}}/{{.Name}}">📁 {{.Name}}/</a>
      {{else}}
        📄 {{.Name}}
      {{end}}
    </td>
    <td class="size">{{if not .IsDir}}{{.Size}}{{end}}</td>
    <td class="time">{{.ModTime}}</td>
    <td>
      {{if .IsDir}}
        <a class="dl-btn" href="/download?path=/{{$.CurrentPath}}/{{.Name}}">下载</a>
      {{end}}
    </td>
  </tr>
  {{end}}
{{else}}
  <tr><td colspan="4" class="empty">此目录为空</td></tr>
{{end}}
</tbody>
</table>
<div class="footer">GoSync v0.1 · 仅局域网使用</div>
</body>
</html>`))

type browseEntry struct {
	Name    string
	Size    string
	ModTime string
	IsDir   bool
}

type browseData struct {
	CurrentPath string
	Parent      string
	Entries     []browseEntry
}

// handleBrowse renders the directory listing for a given path under rootDirs.
func handleBrowse(rootDirs []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/browse/")
		relPath = strings.TrimPrefix(relPath, "/browse")

		// Find which root contains this path
		var targetDir string
		if relPath == "" || relPath == "/" {
			// Show available root directories
			entries := make([]browseEntry, 0, len(rootDirs))
			for _, rd := range rootDirs {
				name := filepath.Base(rd)
				entries = append(entries, browseEntry{
					Name:  name,
					IsDir: true,
				})
			}
			data := browseData{
				CurrentPath: "",
				Entries:     entries,
			}
			browseTmpl.Execute(w, data)
			return
		}

		cleanPath := filepath.FromSlash(strings.TrimPrefix(relPath, "/"))
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
			http.NotFound(w, r)
			return
		}

		// Read directory
		ents, err := os.ReadDir(targetDir)
		if err != nil {
			http.Error(w, "无法读取目录", http.StatusInternalServerError)
			return
		}

		entries := make([]browseEntry, 0, len(ents))
		for _, e := range ents {
			info, err := e.Info()
			if err != nil {
				continue
			}
			entry := browseEntry{
				Name:    e.Name(),
				IsDir:   e.IsDir(),
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
			}
			if !e.IsDir() {
				entry.Size = formatSize(info.Size())
			}
			entries = append(entries, entry)
		}

		// Calculate parent path
		parent := ""
		if dir := filepath.Dir(relPath); dir != "." {
			parent = filepath.ToSlash(dir)
			if parent == "/" {
				parent = ""
			}
		}

		data := browseData{
			CurrentPath: relPath,
			Parent:      parent,
			Entries:     entries,
		}
		browseTmpl.Execute(w, data)
	}
}

func formatSize(n int64) string {
	if n < 1024 {
		return itoa(n) + " B"
	}
	if n < 1024*1024 {
		return itoa(n/1024) + " KB"
	}
	if n < 1024*1024*1024 {
		return itoa(n/(1024*1024)) + " MB"
	}
	return itoa(n/(1024*1024*1024)) + " GB"
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func init() {
	// Ensure embed compatibility if used
	_ = fs.Stat
}

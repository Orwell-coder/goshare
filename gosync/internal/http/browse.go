package http

import (
	"html/template"
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
<title>GoSync{{if .CurrentPath}} — {{.CurrentPath}}{{end}}</title>
<style>
/* === Reset & Base === */
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
body{
  font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,"PingFang SC","Noto Sans SC",sans-serif;
  background:#f3f4f6;color:#111827;min-height:100vh;line-height:1.6;
  -webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale;
}

/* === Layout === */
.container{max-width:1120px;margin:0 auto;padding:24px 20px}

/* === Card === */
.card{background:#fff;border-radius:14px;box-shadow:0 1px 2px rgba(0,0,0,.04);padding:22px 28px;margin-bottom:14px}

/* === Header === */
.header-row{display:flex;align-items:flex-start;gap:14px}
.logo{
  width:42px;height:42px;background:#4f6ef7;border-radius:10px;
  display:flex;align-items:center;justify-content:center;color:#fff;flex-shrink:0;
}
.header-text h1{font-size:20px;font-weight:700;color:#111827;line-height:1.3}
.header-text .sub{font-size:13px;color:#9ca3af;margin-top:1px}

/* === Breadcrumb === */
.crumbs{
  display:flex;align-items:center;gap:2px;margin-top:16px;padding-top:16px;
  border-top:1px solid #f3f4f6;flex-wrap:wrap;
}
.crumbs a{
  display:inline-flex;align-items:center;gap:4px;color:#4f6ef7;
  text-decoration:none;font-size:13px;padding:4px 10px;border-radius:6px;transition:background .15s;
}
.crumbs a:hover{background:#eef1ff}
.crumbs .bc-sep{color:#d1d5db;margin:0 1px}
.crumbs .bc-cur{font-size:13px;color:#374151;font-weight:500;padding:4px 10px}

/* === Toolbar === */
.toolbar{display:flex;align-items:center;gap:8px;margin-bottom:14px;flex-wrap:wrap}
.toolbar .spacer{flex:1}
.btn{
  display:inline-flex;align-items:center;gap:6px;padding:9px 18px;border-radius:8px;
  font-size:13px;font-weight:500;text-decoration:none;cursor:pointer;
  transition:all .15s;line-height:1;white-space:nowrap;
}
.btn svg{flex-shrink:0}
.btn-primary{background:#4f6ef7;color:#fff}
.btn-primary:hover{background:#3d5ce5}
.btn-ghost{background:#fff;color:#4b5563;border:1px solid #e5e7eb}
.btn-ghost:hover{background:#f9fafb;border-color:#d1d5db}

/* === Table === */
.table-wrap{background:#fff;border-radius:14px;box-shadow:0 1px 2px rgba(0,0,0,.04);overflow:hidden}
.table-wrap .t-hint{padding:14px 28px;font-size:13px;color:#9ca3af;border-bottom:1px solid #f3f4f6}
table{width:100%;border-collapse:collapse}
thead th{
  text-align:left;padding:12px 28px;font-size:11px;font-weight:600;color:#9ca3af;
  text-transform:uppercase;letter-spacing:.05em;background:#fafbfc;
  border-bottom:1px solid #f3f4f6;white-space:nowrap;
}
tbody td{padding:13px 28px;border-bottom:1px solid #f9fafb;font-size:14px;vertical-align:middle}
tbody tr:last-child td{border-bottom:none}
tbody tr:hover td{background:#f9fafb}

/* Name column */
.col-name{display:flex;align-items:center;gap:10px}
.col-name .ficon{
  width:34px;height:34px;border-radius:7px;
  display:flex;align-items:center;justify-content:center;flex-shrink:0;
}
.ficon-folder{background:#eef1ff;color:#4f6ef7}
.ficon-file{background:#f3f4f6;color:#9ca3af}
.col-name a{color:#111827;text-decoration:none;font-weight:500;transition:color .15s}
.col-name a:hover{color:#4f6ef7}
.col-name .fn{color:#4b5563}

/* Size & Time */
.col-size{color:#9ca3af;text-align:right;white-space:nowrap;font-variant-numeric:tabular-nums}
.col-time{color:#9ca3af;white-space:nowrap;font-size:13px}

/* Download action */
.dl-link{
  display:inline-flex;align-items:center;gap:4px;padding:6px 14px;border-radius:6px;
  font-size:12px;font-weight:500;color:#4f6ef7;text-decoration:none;
  background:#eef1ff;transition:all .15s;white-space:nowrap;
}
.dl-link:hover{background:#4f6ef7;color:#fff}

/* Empty state */
.empty-row td{text-align:center;padding:56px 28px !important}
.empty-state{color:#d1d5db}
.empty-state svg{display:block;margin:0 auto 12px;opacity:.35}
.empty-state p{font-size:14px}

/* Footer */
.footer{text-align:center;padding:20px;color:#d1d5db;font-size:12px}

/* === Responsive === */
@media(max-width:640px){
  .container{padding:12px}
  .card,.table-wrap{padding:16px 18px;border-radius:10px}
  .toolbar{gap:6px}
  .btn{padding:7px 12px;font-size:12px}
  td,th{padding:10px 16px}
  .col-time{display:none}
}
</style>
</head>
<body>
<div class="container">

  <!-- Header -->
  <div class="card">
    <div class="header-row">
      <div class="logo">
        <svg width="21" height="21" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>
      </div>
      <div class="header-text">
        <h1>GoSync</h1>
        <p class="sub">局域网文件同步</p>
      </div>
    </div>
    {{if not .IsRoot}}{{if .Breadcrumbs}}
    <div class="crumbs">
      <a href="/">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
      </a>
      {{range $i, $c := .Breadcrumbs}}
      <span class="bc-sep">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#d1d5db" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
      </span>
      {{if eq $c.Path $.CurrentPath}}
      <span class="bc-cur">{{$c.Name}}</span>
      {{else}}
      <a href="/browse/{{$c.Path}}">{{$c.Name}}</a>
      {{end}}
      {{end}}
    </div>
    {{end}}{{end}}
  </div>

  <!-- Toolbar -->
  <div class="toolbar">
    <a href="/" class="btn btn-ghost">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
      根目录
    </a>
    {{if .ParentPath}}<a href="/browse/{{.ParentPath}}" class="btn btn-ghost">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
      上级目录
    </a>{{end}}
    <span class="spacer"></span>
    {{if .CurrentPath}}<a href="/download?path=/{{.CurrentPath}}" class="btn btn-primary">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
      下载此文件夹
    </a>{{end}}
  </div>

  <!-- Table -->
  <div class="table-wrap">
    {{if .IsRoot}}<div class="t-hint">共享目录</div>{{end}}
    <table>
      <thead><tr>
        <th>名称</th>
        <th style="text-align:right">大小</th>
        <th>修改时间</th>
        <th style="text-align:center">操作</th>
      </tr></thead>
      <tbody>
      {{if .Entries}}{{range .Entries}}
      <tr>
        <td>
          <div class="col-name">
            {{if .IsDir}}
            <div class="ficon ficon-folder">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>
            </div>
            <a href="/browse/{{$.Breadcrumb}}/{{.Name}}">{{.Name}}</a>
            {{else}}
            <div class="ficon ficon-file">
              <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            </div>
            <span class="fn">{{.Name}}</span>
            {{end}}
          </div>
        </td>
        <td class="col-size">{{if not .IsDir}}{{.Size}}{{end}}</td>
        <td class="col-time">{{.ModTime}}</td>
        <td style="text-align:center">
          <a class="dl-link" href="/download?path=/{{$.Breadcrumb}}/{{.Name}}">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            下载
          </a>
        </td>
      </tr>
      {{end}}{{else}}
      <tr class="empty-row"><td colspan="4">
        <div class="empty-state">
          <svg width="44" height="44" viewBox="0 0 24 24" fill="none" stroke="#d1d5db" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>
          <p>此目录为空</p>
        </div>
      </td></tr>
      {{end}}
      </tbody>
    </table>
  </div>

  <!-- Footer -->
  <div class="footer">GoSync v0.1 · 仅局域网使用</div>

</div>
</body>
</html>`))

type browseEntry struct {
	Name    string
	Size    string
	ModTime string
	IsDir   bool
}

type crumb struct {
	Name string
	Path string
}

type browseData struct {
	CurrentPath string
	ParentPath  string
	Breadcrumb  string
	Breadcrumbs []crumb
	IsRoot      bool
	Entries     []browseEntry
}

// handleBrowse renders directory listing for paths under configured root directories.
// Path format: /browse/<rootName>/<subpath...>
func handleBrowse(rootDirs []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/browse/")
		relPath = strings.TrimPrefix(relPath, "/browse")
		relPath = strings.Trim(relPath, "/")

		if relPath == "" {
			// Root level: show all configured root directories
			entries := make([]browseEntry, 0, len(rootDirs))
			for _, rd := range rootDirs {
				name := filepath.Base(rd)
				entries = append(entries, browseEntry{
					Name:  name,
					IsDir: true,
				})
			}
			browseTmpl.Execute(w, browseData{
				CurrentPath: "",
				Breadcrumb:  "",
				IsRoot:      true,
				Entries:     entries,
			})
			return
		}

		// Build breadcrumbs from path segments
		segments := strings.Split(relPath, "/")
		breadcrumbs := make([]crumb, len(segments))
		for i, seg := range segments {
			breadcrumbs[i] = crumb{
				Name: seg,
				Path: strings.Join(segments[:i+1], "/"),
			}
		}

		// Resolve path: first segment identifies the root directory by name
		parts := strings.SplitN(relPath, "/", 2)
		rootName := parts[0]
		subPath := ""
		if len(parts) > 1 {
			subPath = parts[1]
		}

		// Find the root by name
		var targetDir string
		for _, root := range rootDirs {
			if filepath.Base(root) == rootName {
				if subPath == "" {
					targetDir = root
				} else {
					candidate := filepath.Join(root, filepath.FromSlash(subPath))
					if info, err := os.Stat(candidate); err == nil && info.IsDir() {
						targetDir = candidate
					}
				}
				break
			}
		}

		if targetDir == "" {
			http.NotFound(w, r)
			return
		}

		// Read directory entries
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

		// Build parent path for navigation
		parentPath := ""
		if idx := strings.LastIndex(relPath, "/"); idx >= 0 {
			parentPath = relPath[:idx]
		}

		browseTmpl.Execute(w, browseData{
			CurrentPath: relPath,
			ParentPath:  parentPath,
			Breadcrumb:  relPath,
			Breadcrumbs: breadcrumbs,
			Entries:     entries,
		})
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

package main

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"
)

//go:embed home.html
var gIndexPage string

// renderTemplate 渲染HTML模板
func renderTemplate(w http.ResponseWriter, dirPath string, files []FileInfo) {
	// 计算上一级目录的路径
	parentPath := ""
	if dirPath != "/" {
		filePathDir := filepath.Dir(dirPath)
		// Patch: replace \ to /, compatibility for windows
		filePathDir = strings.ReplaceAll(filePathDir, "\\", "/")
		parentPath = path.Dir(filePathDir)
	}

	t, err := template.New("webpage").Parse(gIndexPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// logger.Printf("renderTemplate dirPath=%s, parentPath=%s, files=%v", dirPath, parentPath, files)

	data := struct {
		DirPath      string
		ParentPath   string
		Files        []FileInfo
		EnableUpload bool
	}{
		DirPath:      dirPath,
		ParentPath:   parentPath,
		Files:        files,
		EnableUpload: *gUpload,
	}

	w.WriteHeader(http.StatusOK)
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

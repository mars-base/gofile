package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func uploadHandleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 处理请求头信息
	// 通过设置 X-Content-Type-Options 响应头，你可以防止浏览器进行 MIME 类型嗅探，并确保内容按照预期的方式被渲染
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	// 检查是否允许上传文件
	if !*gUpload {
		http.Error(w, "upload file is not allowed", http.StatusForbidden)
		return
	}

	// 获取Content-Length头
	contentLength := r.Header.Get("Content-Length")
	if contentLength == "" {
		http.Error(w, "Content-Length header is missing", http.StatusBadRequest)
		return
	}

	// 转换Content-Length值为int64
	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Content-Length", http.StatusBadRequest)
		return
	}

	// 检查大小是否超过限制（这里假设限制是10MB）
	maxSize := int64(*gUploadSize * 1024 * 1024)
	if size > maxSize {
		http.Error(w, fmt.Sprintf("File too big, limit %dMB", maxSize/1024/1024), http.StatusRequestEntityTooLarge)
		return
	}

	// 处理上传的文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.Println("Error retrieving the file:", err)
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 获取上传目标路径并进行规范化
	dirPath := r.URL.Query().Get("dir")
	if dirPath == "" {
		http.Error(w, "Error received dirPath", http.StatusBadRequest)
		return
	}
	// logger.Println(dirPath)
	dirPath = strings.ReplaceAll(strings.ReplaceAll(filepath.Clean(dirPath), "..", ""), "//", "")
	if dirPath == "" {
		dirPath = "/"
	}
	curUrlPath := dirPath
	logger.Println("Uploading file to url:", curUrlPath)

	// 将url路径和服务器绝对路径连接起来
	dirPath = filepath.Join(gAbsPath, dirPath)
	logger.Printf("Uploading file: %s to directory: %s", handler.Filename, dirPath)

	// 创建文件在服务器端保存
	dst, err := os.Create(filepath.Join(dirPath, handler.Filename))
	if err != nil {
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 使用缓冲区进行文件拷贝
	buffer := make([]byte, 1024*1024) // 1MB的缓冲区大小，可以根据实际情况调整
	_, err = io.CopyBuffer(dst, file, buffer)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))

	// 上传成功后刷新当前页面
	// http.Redirect(w, r, curUrlPath, http.StatusSeeOther)
}

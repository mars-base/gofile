package main

import (
	"fmt"
	"gofile/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 打印请求头
func showRequestHeader(r *http.Request) {
	if r.Method == http.MethodGet {
		for key, values := range r.Header {
			for _, value := range values {
				logger.Printf("request header, %s: %s\n", key, value)
			}
		}
	}
}

func faviconHandleFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", http.DetectContentType(gFavBytes))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Length", fmt.Sprint(len(gFavBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(gFavBytes)
}

// 处理文件下载
func rootHandleFunc(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法是否为GET或HEAD
	// logger.Printf("request method: %s, url: %s\n", r.Method, r.URL.Path)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 处理请求头信息
	// 通过设置 X-Content-Type-Options 响应头，你可以防止浏览器进行 MIME 类型嗅探，并确保内容按照预期的方式被渲染
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	// 处理url路径
	// 将url路径和服务器的绝对路径连接起来
	urlPath := strings.ReplaceAll(strings.ReplaceAll(filepath.Clean(r.URL.Path), "..", ""), "//", "")
	filePath := filepath.Join(gAbsPath, urlPath)
	// logger.Println(filePath)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 判断请求是HEAD请求还是GET请求
	// 如果是HEAD请求，只返回响应头信息
	if r.Method == http.MethodHead {
		w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		w.WriteHeader(http.StatusOK)
		return
	}

	// 如果是目录，显示目录中的文件和子目录信息
	// 如果是文件，处理文件下载
	if fileInfo.IsDir() {
		files, err := getFileInfo(filePath, urlPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderTemplate(w, r.URL.Path, files)
	} else {
		// 判断文件是否存在
		if !utils.FileExist(filePath) {
			http.NotFound(w, r)
			return
		}

		// http请求头信息
		if false {
			showRequestHeader(r)
		}

		// 处理文件
		size := fileInfo.Size()
		// 一般性的文件，不使用缓存，直接从文件系统中读取文件，
		// 对于小文件，可以使用缓存文件数据到内存，请求文件下载的时候，直接从cache里获得文件数据返回
		if *gCache && size < int64(*gCacheFileSize*1024*1024) {
			var ref *[]byte
			// logger.Println("Using cache to handle file")
			cv, ok := utils.CacheGetKey(filePath)
			if !ok {
				logger.Println("Cache miss, setting key:", filePath)
				data, err := os.ReadFile(filePath)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				utils.CacheSetKey(filePath, data, 0)
				ref = &data
			} else {
				// logger.Println("Cache hit")
				b := cv.([]byte)
				ref = &b
			}

			// http返回头信息
			w.Header().Set("Content-Type", http.DetectContentType(*ref))
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

			// 检测请求头包含If-Modified-Since字段信息，和服务器本地文件最后修改时间戳进行比对，
			// 如果没有变化，返回304，内容为空即可
			// 304 Not Modified
			since := r.Header.Get("If-Modified-Since")
			// logger.Println("If-Modified-Since:", since)
			// logger.Println("File last modified:", fileInfo.ModTime().UTC().Format(http.TimeFormat))
			if len(since) > 0 {
				ifModifiedSince, _ := time.Parse(time.RFC1123, since)
				if fileInfo.ModTime().Before(ifModifiedSince) {
					// logger.Println("Not modified")
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}

			w.WriteHeader(http.StatusOK)
			WriteBytes(w, ref, 1024*32) // 循环每次写入32KB，防止出现服务器内存暴涨情况
			return
		}

		// 不使用缓存，直接从文件系统中读取文件
		// 使用golang http模块进行文件下载处理
		// http.ServeFile(w, r, filePath)
		// logger.Println("Using cycle read to handle file")
		// logger.Println("file size: ", size)

		// 应对处理大文件，循环读取，每次32k，写入请求返回
		var buffer = make([]byte, 1024*32)
		f, err := os.Open(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// 读取1k字节，用于检测文件类型
		var buf = make([]byte, 1024)
		_, err = f.Read(buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// logger.Println("file type: ", http.DetectContentType(buf))

		// 重置文件指针
		_, err = f.Seek(0, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// http返回头信息
		w.Header().Set("Content-Type", http.DetectContentType(buf))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

		// 检测请求头包含If-Modified-Since字段信息，和服务器本地文件最后修改时间戳进行比对，
		// 如果没有变化，返回304，内容为空即可
		// 304 Not Modified
		since := r.Header.Get("If-Modified-Since")
		if len(since) > 0 {
			ifModifiedSince, _ := time.Parse(time.RFC1123, since)
			if fileInfo.ModTime().Before(ifModifiedSince) {
				// logger.Println("file 304 not modified")
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		// 200 OK
		w.WriteHeader(http.StatusOK)

		// 循环读取文件，每次读取32k，写入请求返回
		// idx := 0
		for {
			n, err := f.Read(buffer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if n == 0 || err == io.EOF {
				break
			}
			WriteBytes(w, &buffer, n)
			// idx++
			// logger.Println("file cycle read idx: ", idx)
		}
		return
	}
}

// getFileInfo 获取目录中的文件和子目录信息
func getFileInfo(dirPath string, urlPath string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		fileSize := formatSize(info.Size())                              // 格式化文件大小
		formattedModTime := info.ModTime().Format("2006-01-02 15:04:05") // 格式化修改时间

		files = append(files, FileInfo{
			Name:             info.Name(),
			Size:             fileSize,
			IsDir:            info.IsDir(),
			Path:             filepath.Join(urlPath, info.Name()),
			FormattedModTime: formattedModTime,
		})
	}

	// 按文件名排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

// formatSize 格式化文件大小为 K、M、G 单位
func formatSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d Bytes", size)
	}
}

func WriteBytes(w http.ResponseWriter, ref *[]byte, step int) {
	idx := 0
	length := len(*ref)
	for {
		if idx+step < length {
			w.Write((*ref)[idx : idx+step])
			idx += step
		} else {
			w.Write((*ref)[idx:])
			break
		}
	}
}

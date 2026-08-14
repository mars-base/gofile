package main

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"gofile/utils"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const chunkSize = 1024 * 32 // 32KB 文件读写块大小

// --- Handlers ---

func faviconHandleFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", http.DetectContentType(gFavBytes))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Length", fmt.Sprint(len(gFavBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(gFavBytes)
}

func rootHandleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	urlPath := strings.ReplaceAll(strings.ReplaceAll(filepath.Clean(r.URL.Path), "..", ""), "//", "")
	filePath := filepath.Join(gAbsPath, urlPath)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodHead {
		w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		w.WriteHeader(http.StatusOK)
		return
	}

	if fileInfo.IsDir() {
		files, err := getFileInfo(filePath, urlPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderTemplate(w, r.URL.Path, files)
	} else {
		if *gCache && fileInfo.Size() < int64(*gCacheFileSize*1024*1024) {
			serveFileCached(w, r, filePath, fileInfo)
		} else {
			serveFileStream(w, r, filePath, fileInfo)
		}
	}
}

func uploadHandleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	if !*gUpload {
		http.Error(w, "upload file is not allowed", http.StatusForbidden)
		return
	}

	contentLength := r.Header.Get("Content-Length")
	if contentLength == "" {
		http.Error(w, "Content-Length header is missing", http.StatusBadRequest)
		return
	}

	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Content-Length", http.StatusBadRequest)
		return
	}

	maxSize := int64(*gUploadSize * 1024 * 1024)
	if size > maxSize {
		http.Error(w, fmt.Sprintf("File too big, limit %dMB", maxSize/1024/1024), http.StatusRequestEntityTooLarge)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.Println("Error retrieving the file:", err)
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	dirPath := r.URL.Query().Get("dir")
	if dirPath == "" {
		http.Error(w, "Error received dirPath", http.StatusBadRequest)
		return
	}
	dirPath = strings.ReplaceAll(strings.ReplaceAll(filepath.Clean(dirPath), "..", ""), "//", "")
	if dirPath == "" {
		dirPath = "/"
	}

	absPath := filepath.Join(gAbsPath, dirPath)
	if !strings.HasPrefix(absPath, gAbsPath) {
		http.Error(w, "Invalid upload path", http.StatusBadRequest)
		return
	}
	dirPath = absPath

	logger.Printf("Uploading file: %s to directory: %s", handler.Filename, dirPath)

	dst, err := os.Create(filepath.Join(dirPath, handler.Filename))
	if err != nil {
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	buffer := make([]byte, 1024*1024)
	_, err = io.CopyBuffer(dst, file, buffer)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}

// --- File serving ---

func serveFileCached(w http.ResponseWriter, r *http.Request, filePath string, fileInfo os.FileInfo) {
	size := fileInfo.Size()
	var ref *[]byte

	cv, ok := utils.CacheGetKey(filePath)
	if !ok {
		data, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		utils.CacheSetKey(filePath, data, 0)
		ref = &data
	} else {
		b := cv.([]byte)
		ref = &b
	}

	w.Header().Set("Content-Type", http.DetectContentType(*ref))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

	if handleNotModified(w, r, fileInfo) {
		return
	}

	w.WriteHeader(http.StatusOK)
	writeBytes(w, ref, chunkSize)
}

func serveFileStream(w http.ResponseWriter, r *http.Request, filePath string, fileInfo os.FileInfo) {
	size := fileInfo.Size()

	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	var buf = make([]byte, 512)
	_, err = f.Read(buf)
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", http.DetectContentType(buf))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

	if handleNotModified(w, r, fileInfo) {
		return
	}

	w.WriteHeader(http.StatusOK)

	var buffer = make([]byte, chunkSize)
	for {
		n, err := f.Read(buffer)
		if n > 0 {
			writeBytes(w, &buffer, n)
		}
		if err != nil {
			if err != io.EOF {
				logger.Printf("error reading file %s: %v", filePath, err)
			}
			break
		}
	}
}

func handleNotModified(w http.ResponseWriter, r *http.Request, fileInfo os.FileInfo) bool {
	since := r.Header.Get("If-Modified-Since")
	if len(since) > 0 {
		ifModifiedSince, _ := time.Parse(time.RFC1123, since)
		if fileInfo.ModTime().Before(ifModifiedSince) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}

func writeBytes(w http.ResponseWriter, ref *[]byte, step int) {
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

// --- Directory listing ---

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
		files = append(files, FileInfo{
			Name:             info.Name(),
			Size:             formatSize(info.Size()),
			IsDir:            info.IsDir(),
			Path:             filepath.Join(urlPath, info.Name()),
			FormattedModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

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

// --- Template rendering ---

func renderTemplate(w http.ResponseWriter, dirPath string, files []FileInfo) {
	parentPath := ""
	if dirPath != "/" {
		filePathDir := filepath.Dir(dirPath)
		filePathDir = strings.ReplaceAll(filePathDir, "\\", "/")
		parentPath = filepath.Dir(filePathDir)
	}

	t, err := template.New("webpage").Parse(gIndexPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

// --- Basic Auth ---

func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized\n"))
			return
		}

		authParts := strings.SplitN(auth, " ", 2)
		if len(authParts) != 2 || authParts[0] != "Basic" {
			http.Error(w, "Bad authorization header", http.StatusBadRequest)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(authParts[1])
		if err != nil {
			http.Error(w, "Bad authorization header", http.StatusBadRequest)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if !checkAuth(pair) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized\n"))
			return
		}

		next.ServeHTTP(w, r)
	}
}

func checkAuth(pair []string) bool {
	if len(pair) != 2 {
		return false
	}
	for _, auth := range gBasicAuthList {
		if pair[0] == auth[0] && pair[1] == auth[1] {
			return true
		}
	}
	return false
}

// --- Logging middleware (with gzip) ---

type loggingResponseWriter struct {
	http.ResponseWriter
	Writer        *gzip.Writer
	statusCode    int
	headerWritten bool
	isGzip        bool
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if !lrw.headerWritten {
		if lrw.isGzip {
			lrw.ResponseWriter.Header().Del("Content-Length")
		}
		lrw.statusCode = code
		lrw.headerWritten = true
		lrw.ResponseWriter.WriteHeader(code)
	}
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.isGzip {
		return lrw.Writer.Write(b)
	}
	return lrw.ResponseWriter.Write(b)
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w}

		if *gOpenGzip && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gz := gzip.NewWriter(w)
			defer gz.Close()

			w.Header().Del("Content-Length")
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			lrw.Writer = gz
			lrw.isGzip = true
		}

		handler.ServeHTTP(lrw, r)
		logger.Printf(
			"%s - %s %s %s %d %s",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			r.Proto,
			lrw.statusCode,
			time.Since(start),
		)
	})
}

package main

import (
	"compress/gzip"
	"net/http"
	"strings"
	"time"
)

// logRequest 是一个记录请求信息的中间件
func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w}

		// 检查客户端是否支持 gzip 压缩
		// 会引起除文本这类容易压缩的文件之外的文件传输速率下降，服务器的cpu升高
		// 好处是同样的带宽可以传输文件数量增多
		if *gOpenGzip && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// 创建 gzip.Writer
			gz := gzip.NewWriter(w)
			defer gz.Close()

			// 设置响应头
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			// 使用自定义的 ResponseWriter 以便写入压缩数据
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

// loggingResponseWriter 是一个自定义的 ResponseWriter，用于记录状态码
type loggingResponseWriter struct {
	http.ResponseWriter
	Writer        *gzip.Writer
	statusCode    int
	headerWritten bool
	isGzip        bool
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if !lrw.headerWritten {
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

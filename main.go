package main

// Time    :   2024-06-24 04:31:17 PM
// Author  :   diwen

import (
	"context"
	"flag"
	"gofile/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const ()

var (
	logger = utils.Logger // 日志对象
)

// cmdline parameter
var gListenIp *string         // 监听IP
var gPort *string             // 端口
var gDir *string              // 静态文件目录
var gAbsPath string           // 绝对路径
var gShowVersion *bool        // 是否显示版本信息
var gOpenGzip *bool           // 是否开启gzip压缩
var gUpload *bool             // 是否允许上传
var gUploadSize *int          // 允许上传的文件大小 xMB
var gCert *string             // 证书文件路径
var gKey *string              // 私钥文件路径
var gCache *bool              // 是否开启cache
var gCacheFileSize *int       // 缓存文件大小 默认1M
var gCacheTime *int           // 缓存时间 x分钟
var gDoc *bool                // 是否显示文档
var gAuth *bool               // 是否开启basic auth
var gBasicAuthString *string  // basic auth string, 格式：user1:passwd1,user2:passwd2
var gBasicAuthList [][]string // basic auth list, 格式：[["user1","passwd1"],["user2","passwd2"]]

// 定义文件信息结构体
type FileInfo struct {
	Name             string // 文件名
	Size             string // 使用字符串类型来存储格式化后的文件大小
	IsDir            bool   // 是否是目录
	Path             string // 文件路径
	FormattedModTime string // 格式化后的修改时间字符串
}

func main() {
	// 定义命令行参数
	gListenIp = flag.String("h", "0.0.0.0", "set listen ip, 127.0.0.1/localhost/0.0.0.0")
	gPort = flag.String("p", "8080", "set server's port")
	gDir = flag.String("d", "./", "set static file directory")
	gShowVersion = flag.Bool("v", false, "show version")
	gOpenGzip = flag.Bool("gzip", false, "Is open gzip to transport file.")
	gUpload = flag.Bool("upload", false, "Is allow upload file.")
	gUploadSize = flag.Int("uploadSize", 10, "Allow upload file size xMB")
	gCert = flag.String("cert", "", "set certificate file path")
	gKey = flag.String("key", "", "set private key file path")
	gCache = flag.Bool("cache", false, "Is open cache to transport file.")
	gCacheFileSize = flag.Int("cacheSize", 1, "Cache file size xMB")
	gCacheTime = flag.Int("cacheTime", 10, "Cache file time x minutes")
	gDoc = flag.Bool("doc", false, "Is show document.")
	gAuth = flag.Bool("auth", false, "Is open basic auth.")
	gBasicAuthString = flag.String("authString", "admin:admin", "basic auth string, format: user1:passwd1,user2:passwd2")
	flag.Parse()

	if *gShowVersion {
		logger.Println(gName + " - " + gVersion)
		os.Exit(0)
	}

	if *gDoc {
		logger.Println(gUsage)
		os.Exit(0)

		logger.Println("Upload file")
		os.Exit(0)
	}

	if len(*gPort) == 0 || len(*gDir) == 0 {
		logger.Fatalf("parameter invalid, -h for help")
	}

	if !utils.DirExist(*gDir) {
		logger.Fatalln("static directory path [", *gDir, "] not exist. -h for help")
	}

	if *gCert != "" || *gKey != "" {
		if !utils.FileExist(*gCert) {
			logger.Fatalln("certificate file path [", *gCert, "] not exist. -h for help")
		}
		if !utils.FileExist(*gKey) {
			logger.Fatalln("private key file path [", *gKey, "] not exist. -h for help")
		}
	}

	// 转换为绝对路径
	gAbsPath = *gDir
	gAbsPath, _ = filepath.Abs(gAbsPath)

	// 创建cache，缓存文件功能
	utils.CacheCreate(*gCacheTime*60, *gCacheTime*2*60) // 缓存x分钟，2倍x分钟后自动清理回收内存

	// 创建路由
	hander := http.NewServeMux()

	// favicon url
	hander.Handle("/favicon.png", logRequest(http.HandlerFunc(faviconHandleFunc)))

	if !*gAuth {
		// 文件上传
		hander.Handle("/upload", logRequest(http.HandlerFunc(uploadHandleFunc)))
		// 文件处理
		hander.Handle("/", logRequest(http.HandlerFunc(rootHandleFunc)))
	} else {
		hander.Handle("/upload", logRequest(http.HandlerFunc(BasicAuth(uploadHandleFunc))))
		hander.Handle("/", logRequest(http.HandlerFunc(BasicAuth(rootHandleFunc))))
	}

	// Create a new HTTP server
	gServer := &http.Server{
		Addr:    *gListenIp + ":" + *gPort,
		Handler: hander,
		// ReadTimeout:  10 * time.Second,    // 读取超时时间， 不要设定 防止客户端请求数据过程中被强制关闭连接 导致客户端无法下载文件
		// WriteTimeout: 10 * time.Second,    // 写入超时时间
	}

	// 输出提示信息
	logger.Println(gName)
	logger.Printf("Serving directory [%s], HTTP File Server: %s:%s\n", gAbsPath, *gListenIp, *gPort)
	logger.Println("open gzip compress to", *gOpenGzip)
	logger.Println("open upload file to", *gUpload)
	logger.Println("open basic auth to", *gAuth)
	if *gUpload {
		logger.Println("upload file size", *gUploadSize, "MB")
	}
	logger.Println("open cache to", *gCache)
	if *gCache {
		logger.Println("cache file size", *gCacheFileSize, "MB")
		logger.Println("cache file time", *gCacheTime, "minutes")
	}
	if *gAuth {
		// logger.Println("basic auth string", *gBasicAuthString)
		// 解析basic auth string
		authList := utils.Split(*gBasicAuthString, ",")
		// logger.Println("basic auth list", authList)
		for _, auth := range authList {
			auth := utils.Split(auth, ":")
			if len(auth) != 2 {
				logger.Fatalln("basic auth string format error")
			}
			gBasicAuthList = append(gBasicAuthList, auth)
		}
		logger.Println("basic auth list", gBasicAuthList)
	}

	// 启动 HTTP 服务器
	go func() {
		if *gCert != "" && *gKey != "" {
			logger.Println("open https server")
			logger.Fatal(gServer.ListenAndServeTLS(*gCert, *gKey))
		} else {
			logger.Fatal(gServer.ListenAndServe())
		}
	}()

	// exit and clean
	utils.AddGracefulExit(func() error {
		logger.Println("Server shutdown")

		// The context is used to inform the server it has 2 seconds to finish
		// the request it is currently handling, if no request, exit directly
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := gServer.Shutdown(ctx); err != nil {
			logger.Println("server Shutdown:", err)
		}

		utils.SleepSec(1)
		return nil
	})
}

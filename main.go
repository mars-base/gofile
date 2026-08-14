package main

import (
	"context"
	"flag"
	"gofile/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	logger = utils.Logger
)

// --- 命令行参数 ---
var (
	gListenIp        *string
	gPort            *string
	gDir             *string
	gAbsPath         string
	gShowVersion     *bool
	gOpenGzip        *bool
	gUpload          *bool
	gUploadSize      *int
	gCert            *string
	gKey             *string
	gCache           *bool
	gCacheFileSize   *int
	gCacheTime       *int
	gDoc             *bool
	gAuth            *bool
	gBasicAuthString *string
	gBasicAuthList   [][]string
)

// --- 数据结构 ---

type FileInfo struct {
	Name             string
	Size             string
	IsDir            bool
	Path             string
	FormattedModTime string
}

// --- main ---

func main() {
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

	gAbsPath, _ = filepath.Abs(*gDir)
	utils.CacheCreate(*gCacheTime*60, *gCacheTime*2*60)

	hander := http.NewServeMux()
	hander.Handle("/favicon.png", logRequest(http.HandlerFunc(faviconHandleFunc)))

	if !*gAuth {
		hander.Handle("/upload", logRequest(http.HandlerFunc(uploadHandleFunc)))
		hander.Handle("/", logRequest(http.HandlerFunc(rootHandleFunc)))
	} else {
		hander.Handle("/upload", logRequest(http.HandlerFunc(BasicAuth(uploadHandleFunc))))
		hander.Handle("/", logRequest(http.HandlerFunc(BasicAuth(rootHandleFunc))))
	}

	gServer := &http.Server{
		Addr:              *gListenIp + ":" + *gPort,
		Handler:           hander,
		ReadHeaderTimeout: 10 * time.Second,
	}

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
		authList := utils.Split(*gBasicAuthString, ",")
		for _, auth := range authList {
			auth := utils.Split(auth, ":")
			if len(auth) != 2 {
				logger.Fatalln("basic auth string format error")
			}
			gBasicAuthList = append(gBasicAuthList, auth)
		}
		logger.Println("basic auth list", gBasicAuthList)
	}

	go func() {
		if *gCert != "" && *gKey != "" {
			logger.Println("open https server")
			logger.Fatal(gServer.ListenAndServeTLS(*gCert, *gKey))
		} else {
			logger.Fatal(gServer.ListenAndServe())
		}
	}()

	utils.AddGracefulExit(func() error {
		logger.Println("Server shutdown")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := gServer.Shutdown(ctx); err != nil {
			logger.Println("server Shutdown:", err)
		}
		utils.SleepSec(1)
		return nil
	})
}

package main

import _ "embed"

//go:embed home.html
var gIndexPage string

//go:embed favicon.png
var gFavBytes []byte

var gName = "gofile - Simple and flexible file server"
var gVersion = "dev" // 通过 -ldflags "-X main.gVersion=vX.X.X" 注入 git tag 版本
var gUsage = `
Support:
	1. List files with set directory
	2. Download file
	3. Basic auth browser
	4. Upload file
	5. Set upload file size limit
	6. Upload file by curl with basic auth
	7. Https server with self signed certificate
	8. Cache function
	9. Set cache time
	10. Set cache file size limit
	11. Gzip compress function
	12. Big file upload by set upload size

Usage:
	./gofile -h <listen ip> -p <port> -d <path-of-file-directory>
Upload file by curl:
	curl -F "file=@/path/to/file" "http://127.0.0.1:8080/upload?dir=<path-of-file-directory>"
Example:
	curl -F "file=@t.yaml" "http://127.0.0.1:8080/upload?dir=/"
	curl -F "file=@t.yaml" "http://127.0.0.1:8080/upload?dir=/" -u "<username>:<password>"
`

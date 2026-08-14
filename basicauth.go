package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		// logger.Println("auth: ", auth)
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized\n"))
			return
		}

		authParts := strings.SplitN(auth, " ", 2)
		// logger.Println("authParts: ", authParts)
		if len(authParts) != 2 || authParts[0] != "Basic" {
			http.Error(w, "Bad authorization header", http.StatusBadRequest)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(authParts[1])
		// logger.Println("payload: ", payload)
		if err != nil {
			http.Error(w, "Bad authorization header", http.StatusBadRequest)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		// logger.Println("pair: ", pair)
		if !checkAuth(pair) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized\n"))
			return
		}

		next.ServeHTTP(w, r)
	}
}

// 校验用户名和密码
func checkAuth(pair []string) bool {
	if len(pair) != 2 {
		return false
	}

	// 遍历用户名和密码的配置列表
	for _, auth := range gBasicAuthList {
		if pair[0] == auth[0] && pair[1] == auth[1] {
			return true
		}
	}

	return false
}

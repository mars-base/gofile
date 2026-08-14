BUILD_DIR := build
APP_NAME := gofile
PLATFORMS := windows linux darwin-amd64 darwin-arm64
WINDOWS_OUTPUT := $(BUILD_DIR)/$(APP_NAME).exe
LINUX_OUTPUT := $(BUILD_DIR)/$(APP_NAME)-linux
MACOS_AMD64_OUTPUT := $(BUILD_DIR)/$(APP_NAME)-darwin-amd64
MACOS_ARM64_OUTPUT := $(BUILD_DIR)/$(APP_NAME)-darwin-arm64
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X 'main.gVersion=$(VERSION)'

# 默认目标
.PHONY: all
all: $(PLATFORMS)

# 创建build目录
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: dev
dev:
	CompileDaemon \
	-graceful-kill=true \
	-exclude-dir=".git,build" \
	-pattern=".*\.go" \
	-color=true \
	-build="make linux" -command="./$(LINUX_OUTPUT)"

# Windows目标
.PHONY: windows
windows: $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags '$(LDFLAGS)' -o $(WINDOWS_OUTPUT)

# Linux目标
.PHONY: linux
linux: $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '$(LDFLAGS)' -o $(LINUX_OUTPUT)
	@which upx >/dev/null 2>&1 && upx $(LINUX_OUTPUT) || echo "upx not found, skipping compression"

# macOS目标 (amd64)
.PHONY: darwin-amd64
darwin-amd64: $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -ldflags '$(LDFLAGS)' -o $(MACOS_AMD64_OUTPUT)

# macOS目标 (arm64)
.PHONY: darwin-arm64
darwin-arm64: $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -ldflags '$(LDFLAGS)' -o $(MACOS_ARM64_OUTPUT)

# 清理构建文件
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# gofile

Simple and flexible file server with directory listing, upload, and basic auth.

## Features

- Directory listing with file size and modification time
- File download with streaming (supports large files)
- File upload with size limit
- Basic authentication with multiple users
- HTTP caching for small files
- Gzip compression
- HTTPS support with self-signed certificates
- 304 Not Modified support
- Graceful shutdown

## Installation

### Quick Install

**Linux / macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/mars-base/gofile/main/scripts/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/mars-base/gofile/main/scripts/install.ps1 | iex
```

### Build from source
```bash
git clone https://github.com/mars-base/gofile.git
cd gofile
make linux    # or make windows, make darwin-amd64, make darwin-arm64
```

## Usage

```bash
# Start server with default settings (port 8080, current directory)
./gofile

# Custom port and directory
./gofile -p 9000 -d /path/to/files

# Enable upload (10MB limit)
./gofile -upload -uploadSize 10

# Enable basic auth
./gofile -auth -authString "admin:pass123,user2:pass456"

# Enable caching for files under 1MB
./gofile -cache -cacheSize 1 -cacheTime 10

# Enable gzip compression
./gofile -gzip

# HTTPS with certificates
./gofile -cert cert.pem -key key.pem

# Full example
./gofile -p 8080 -d ./files -upload -uploadSize 50 -auth -authString "admin:secret" -cache -gzip
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-h` | `0.0.0.0` | Listen IP |
| `-p` | `8080` | Server port |
| `-d` | `./` | Static file directory |
| `-v` | - | Show version |
| `-doc` | - | Show help |
| `-gzip` | `false` | Enable gzip compression |
| `-upload` | `false` | Enable file upload |
| `-uploadSize` | `10` | Upload size limit (MB) |
| `-cache` | `false` | Enable file caching |
| `-cacheSize` | `1` | Cache file size limit (MB) |
| `-cacheTime` | `10` | Cache expiration time (minutes) |
| `-auth` | `false` | Enable basic authentication |
| `-authString` | `admin:admin` | Auth credentials (format: `user1:pass1,user2:pass2`) |
| `-cert` | - | TLS certificate file path |
| `-key` | - | TLS private key file path |

## Upload via curl

```bash
# Without auth
curl -F "file=@/path/to/file" "http://localhost:8080/upload?dir=/"

# With auth
curl -F "file=@/path/to/file" "http://localhost:8080/upload?dir=/" -u "admin:pass123"
```

## License

MIT

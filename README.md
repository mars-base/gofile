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

### Deploy with Ansible
```bash
# Install roles and playbooks into your project
bash ansible/install.sh

# Deploy to servers (default config)
ansible-playbook -i hosts.ini playbooks/gofile.yml -e "HOSTS=servers"

# Deploy with custom configuration
ansible-playbook -i hosts.ini playbooks/gofile.yml \
  -e "HOSTS=servers" \
  -e "gofile_port=9000" \
  -e "gofile_upload=true" \
  -e "gofile_upload_size=50" \
  -e "gofile_auth=true" \
  -e "gofile_auth_string=admin:secret123" \
  -e "gofile_gzip=true"
```

**Ansible variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `gofile_dir` | `/srv/gofile` | Installation directory |
| `gofile_version` | `latest` | Release version (e.g., `v1.0.0`) |
| `gofile_port` | `8080` | Server port |
| `gofile_serve_dir` | `/srv/gofile/files` | Static file directory |
| `gofile_gzip` | `false` | Enable gzip compression |
| `gofile_upload` | `false` | Enable file upload |
| `gofile_upload_size` | `10` | Upload size limit (MB) |
| `gofile_cache` | `false` | Enable file caching |
| `gofile_auth` | `false` | Enable basic authentication |
| `gofile_auth_string` | `admin:admin` | Auth credentials |
| `gofile_cert` | - | TLS certificate file path |
| `gofile_key` | - | TLS private key file path |

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

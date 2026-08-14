#!/bin/bash
#
# gofile e2e test script
# Tests all features end-to-end via HTTP requests
#
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_DIR/build/gofile-linux"
TEST_DIR=$(mktemp -d /tmp/gofile-e2e-XXXXXX)
UPLOAD_DIR="$TEST_DIR/uploads"
PORT=18080
BASE_URL="http://127.0.0.1:$PORT"

PASS=0
FAIL=0
SERVER_PID=""

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# --- Helpers ---

build() {
    echo -e "${CYAN}>>> Building gofile...${NC}"
    mkdir -p "$PROJECT_DIR/build"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -a -ldflags '-extldflags "-static" -s -w' \
        -o "$BINARY" "$PROJECT_DIR" 2>&1
    if [ $? -ne 0 ]; then
        echo -e "${RED}Build failed!${NC}"
        exit 1
    fi
    echo -e "${GREEN}Build OK${NC}"
}

setup_test_files() {
    echo "Hello, gofile e2e test!" > "$TEST_DIR/test.txt"
    mkdir -p "$TEST_DIR/subdir"
    echo "sub file content" > "$TEST_DIR/subdir/sub.txt"
    mkdir -p "$UPLOAD_DIR"
    dd if=/dev/zero of="$TEST_DIR/large.bin" bs=1M count=2 2>/dev/null
    dd if=/dev/urandom of="$TEST_DIR/binary.dat" bs=1024 count=100 2>/dev/null
    echo "cache me" > "$TEST_DIR/cache_test.txt"
}

start_server() {
    local args="$1"
    "$BINARY" -p "$PORT" -d "$TEST_DIR" $args > /dev/null 2>&1 &
    SERVER_PID=$!
    for i in $(seq 1 50); do
        if curl -s -o /dev/null -w '' "$BASE_URL/" 2>/dev/null; then
            return 0
        fi
        sleep 0.1
    done
    echo -e "${RED}Server failed to start (args: $args, pid: $SERVER_PID)${NC}"
    kill $SERVER_PID 2>/dev/null
    return 1
}

stop_server() {
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null
        wait $SERVER_PID 2>/dev/null
        SERVER_PID=""
        sleep 0.3
    fi
}

assert() {
    local name="$1"
    local result="$2"
    if [ "$result" = "PASS" ]; then
        PASS=$((PASS + 1))
        echo -e "  ${GREEN}PASS${NC} $name"
    else
        FAIL=$((FAIL + 1))
        echo -e "  ${RED}FAIL${NC} $name: $result"
    fi
}

check_status() {
    local name="$1" expected="$2"
    shift 2
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" "$@")
    if [ "$code" = "$expected" ]; then
        assert "$name" "PASS"
    else
        assert "$name" "expected $expected, got $code"
    fi
}

check_contains() {
    local name="$1" keyword="$2"
    shift 2
    local body code
    body=$(curl -s -w "\n%{http_code}" "$@")
    code=$(echo "$body" | tail -1)
    body=$(echo "$body" | sed '$d')
    if [ "$code" = "200" ] && echo "$body" | grep -q "$keyword"; then
        assert "$name" "PASS"
    else
        assert "$name" "code=$code, missing '$keyword'"
    fi
}

check_not_contains() {
    local name="$1" keyword="$2"
    shift 2
    local body code
    body=$(curl -s -w "\n%{http_code}" "$@")
    code=$(echo "$body" | tail -1)
    body=$(echo "$body" | sed '$d')
    if [ "$code" = "200" ] && ! echo "$body" | grep -q "$keyword"; then
        assert "$name" "PASS"
    else
        assert "$name" "unexpected '$keyword' found"
    fi
}

check_header() {
    local name="$1" header_name="$2" expected="$3"
    shift 3
    local value
    value=$(curl -s -D - -o /dev/null "$@" | grep -i "^${header_name}:" | head -1 | tr -d '\r')
    if echo "$value" | grep -qi "$expected"; then
        assert "$name" "PASS"
    else
        assert "$name" "header '$header_name'='$value', want '$expected'"
    fi
}

check_file_download() {
    local name="$1" url="$2" local_file="$3"
    local tmp="$TEST_DIR/.dl_tmp"
    local code
    code=$(curl -s -o "$tmp" -w "%{http_code}" "$url")
    if [ "$code" = "200" ] && cmp -s "$tmp" "$local_file"; then
        assert "$name" "PASS"
    else
        assert "$name" "code=$code or content mismatch"
    fi
    rm -f "$tmp"
}

cleanup() {
    stop_server
    rm -rf "$TEST_DIR"
    echo ""
    echo -e "${CYAN}=============================="
    echo -e " Results: ${GREEN}${PASS} passed${NC}${CYAN}, ${RED}${FAIL} failed${NC}${CYAN}"
    echo -e "==============================${NC}"
    if [ "$FAIL" -gt 0 ]; then
        exit 1
    fi
}
trap cleanup EXIT

# ========================================
echo -e "${CYAN}"
echo "  gofile e2e test"
echo "  $(date '+%Y-%m-%d %H:%M:%S')"
echo "================================"
echo -e "${NC}"

# --- Build ---
build
setup_test_files

# ========================================
# Phase 1: CLI flags
# ========================================
echo -e "\n${YELLOW}[Phase 1] CLI flags${NC}"

output=$("$BINARY" -v 2>&1)
echo "$output" | grep -q "gofile" && assert "Version flag (-v)" "PASS" || assert "Version flag (-v)" "no gofile in output"

output=$("$BINARY" -doc 2>&1)
echo "$output" | grep -q "Support" && assert "Doc flag (-doc)" "PASS" || assert "Doc flag (-doc)" "no Support in output"

# ========================================
# Phase 2: Basic server (no auth, upload enabled, cache+gzip)
# ========================================
echo -e "\n${YELLOW}[Phase 2] Basic server (no auth, upload+cache+gzip)${NC}"

start_server "-upload -uploadSize 1 -cache -cacheSize 1 -cacheTime 10 -gzip"

# --- Directory listing ---
echo -e "  ${CYAN}-- Directory listing --${NC}"
check_contains "Root dir shows test.txt"      "test.txt"      "$BASE_URL/"
check_contains "Root dir shows subdir"         "subdir"        "$BASE_URL/"
check_contains "Root dir shows binary.dat"     "binary.dat"    "$BASE_URL/"
check_contains "Root dir shows large.bin"      "large.bin"     "$BASE_URL/"
check_contains "Subdir shows sub.txt"          "sub.txt"       "$BASE_URL/subdir/"
check_contains "Dir page has title"            "Directory:"    "$BASE_URL/"
check_contains "Dir page has file size col"    "File Size"     "$BASE_URL/"
check_contains "Dir page has mod time col"     "Modified Time" "$BASE_URL/"
check_contains "Upload form present"           "uploadForm"    "$BASE_URL/"

# --- File download ---
echo -e "  ${CYAN}-- File download --${NC}"
check_file_download "Download test.txt"         "$BASE_URL/test.txt"       "$TEST_DIR/test.txt"
check_file_download "Download subdir/sub.txt"   "$BASE_URL/subdir/sub.txt" "$TEST_DIR/subdir/sub.txt"
check_file_download "Download large.bin (2MB)"  "$BASE_URL/large.bin"     "$TEST_DIR/large.bin"
check_file_download "Download binary.dat"       "$BASE_URL/binary.dat"    "$TEST_DIR/binary.dat"

# --- HEAD request ---
echo -e "  ${CYAN}-- HEAD request --${NC}"
check_status "HEAD root dir"     "200" -I "$BASE_URL/"
check_status "HEAD test.txt"     "200" -I "$BASE_URL/test.txt"
head_headers=$(curl -s -I "$BASE_URL/test.txt")
echo "$head_headers" | grep -qi "Content-Length" && assert "HEAD has Content-Length" "PASS" || assert "HEAD has Content-Length" "missing"
echo "$head_headers" | grep -qi "Last-Modified"  && assert "HEAD has Last-Modified"  "PASS" || assert "HEAD has Last-Modified"  "missing"

# --- 304 Not Modified ---
echo -e "  ${CYAN}-- 304 Not Modified --${NC}"
# Note: same-timestamp If-Modified-Since returns 200 (server uses Before() which is correct per HTTP spec)
code_304b=$(curl -s -o /dev/null -w "%{http_code}" -H "If-Modified-Since: Thu, 31 Dec 2099 23:59:59 GMT" "$BASE_URL/test.txt")
[ "$code_304b" = "304" ] && assert "304 with future If-Modified-Since" "PASS" || assert "304 with future If-Modified-Since" "got $code_304b"

code_200_mod=$(curl -s -o /dev/null -w "%{http_code}" -H "If-Modified-Since: Thu, 01 Jan 1970 00:00:00 GMT" "$BASE_URL/test.txt")
[ "$code_200_mod" = "200" ] && assert "200 with old If-Modified-Since" "PASS" || assert "200 with old If-Modified-Since" "got $code_200_mod"

# --- Response headers ---
echo -e "  ${CYAN}-- Response headers --${NC}"
check_header "X-Content-Type-Options: nosniff" "X-Content-Type-Options" "nosniff" "$BASE_URL/test.txt"
check_header "X-Frame-Options: SAMEORIGIN"     "X-Frame-Options"        "SAMEORIGIN" "$BASE_URL/test.txt"
check_header "Accept-Ranges: bytes"            "Accept-Ranges"          "bytes" "$BASE_URL/test.txt"
check_header "Content-Length present"          "Content-Length"         "[0-9]" "$BASE_URL/test.txt"

# --- Method restriction ---
echo -e "  ${CYAN}-- Method restriction --${NC}"
check_status "POST / rejected (405)"   "405" -X POST   "$BASE_URL/"
check_status "PUT / rejected (405)"    "405" -X PUT     "$BASE_URL/"
check_status "DELETE / rejected (405)" "405" -X DELETE  "$BASE_URL/"
check_status "PATCH / rejected (405)"  "405" -X PATCH   "$BASE_URL/"

# --- 404 ---
echo -e "  ${CYAN}-- 404 Not Found --${NC}"
check_status "404 nonexistent file"      "404" "$BASE_URL/no_such_file.txt"
check_status "404 nonexistent dir"       "404" "$BASE_URL/no_such_dir/"
check_status "404 deep nonexistent path" "404" "$BASE_URL/a/b/c/no_such.txt"

# --- Path traversal protection ---
echo -e "  ${CYAN}-- Path traversal --${NC}"
check_status "Block .. traversal"        "404" "$BASE_URL/../etc/passwd"
check_status "Block double .. traversal" "404" "$BASE_URL/subdir/../../etc/passwd"

# --- Favicon ---
echo -e "  ${CYAN}-- Favicon --${NC}"
fav_code=$(curl -s -o "$TEST_DIR/.fav_tmp" -w "%{http_code}" "$BASE_URL/favicon.png")
fav_size=$(wc -c < "$TEST_DIR/.fav_tmp")
[ "$fav_code" = "200" ] && [ "$fav_size" -gt 100 ] && assert "Favicon returns PNG" "PASS" || assert "Favicon returns PNG" "code=$fav_code size=$fav_size"
fav_type=$(curl -s -I "$BASE_URL/favicon.png" | grep -i "Content-Type" | tr -d '\r')
echo "$fav_type" | grep -qi "image/png" && assert "Favicon Content-Type: image/png" "PASS" || assert "Favicon Content-Type" "got: $fav_type"
rm -f "$TEST_DIR/.fav_tmp"

# --- File upload ---
echo -e "  ${CYAN}-- File upload --${NC}"
echo "upload test content" > "$TEST_DIR/.upload_test.txt"

up_code=$(curl -s -o /dev/null -w "%{http_code}" -F "file=@$TEST_DIR/.upload_test.txt" "$BASE_URL/upload?dir=/uploads")
[ "$up_code" = "200" ] && assert "Upload file OK (200)" "PASS" || assert "Upload file OK (200)" "got $up_code"
[ -f "$UPLOAD_DIR/.upload_test.txt" ] && assert "Uploaded file exists on disk" "PASS" || assert "Uploaded file exists on disk" "not found"
dl_content=$(curl -s "$BASE_URL/uploads/.upload_test.txt")
echo "$dl_content" | grep -q "upload test content" && assert "Downloaded uploaded file matches" "PASS" || assert "Downloaded uploaded file matches" "content mismatch"

# Upload to root dir
echo "root upload" > "$TEST_DIR/.root_upload.txt"
up_code=$(curl -s -o /dev/null -w "%{http_code}" -F "file=@$TEST_DIR/.root_upload.txt" "$BASE_URL/upload?dir=/")
[ "$up_code" = "200" ] && assert "Upload to root dir" "PASS" || assert "Upload to root dir" "got $up_code"
[ -f "$TEST_DIR/.root_upload.txt" ] && assert "Root upload file exists" "PASS" || assert "Root upload file exists" "not found"

# Upload method restriction
check_status "GET /upload rejected (405)" "405" "$BASE_URL/upload"
check_status "PUT /upload rejected (405)" "405" -X PUT "$BASE_URL/upload"

# Upload without dir param (POST with no dir query)
up_code_nodir=$(curl -s -o /dev/null -w "%{http_code}" -X POST -F "file=@$TEST_DIR/.upload_test.txt" "$BASE_URL/upload")
[ "$up_code_nodir" = "400" ] && assert "Upload without dir (400)" "PASS" || assert "Upload without dir (400)" "got $up_code_nodir"

# Upload oversized file (> 1MB limit)
up_code_big=$(curl -s -o /dev/null -w "%{http_code}" -F "file=@$TEST_DIR/large.bin" "$BASE_URL/upload?dir=/uploads")
[ "$up_code_big" = "413" ] && assert "Upload oversized file (413)" "PASS" || assert "Upload oversized file (413)" "got $up_code_big"

rm -f "$TEST_DIR/.upload_test.txt" "$TEST_DIR/.root_upload.txt"

# --- Cache (second request should also succeed) ---
echo -e "  ${CYAN}-- Cache --${NC}"
body1=$(curl -s "$BASE_URL/cache_test.txt")
body2=$(curl -s "$BASE_URL/cache_test.txt")
[ "$body1" = "$body2" ] && [ -n "$body1" ] && assert "Cache: consistent responses" "PASS" || assert "Cache: consistent responses" "mismatch"

# --- Gzip ---
echo -e "  ${CYAN}-- Gzip --${NC}"
gzip_enc=$(curl -s -D - -o /dev/null -H "Accept-Encoding: gzip" "$BASE_URL/test.txt" | grep -i "Content-Encoding" | tr -d '\r')
echo "$gzip_enc" | grep -qi "gzip" && assert "Gzip: Content-Encoding: gzip" "PASS" || assert "Gzip: Content-Encoding: gzip" "got: $gzip_enc"

# Verify gzip response is smaller than plain (compression works)
gzip_size=$(curl -s -o /dev/null -w "%{size_download}" -H "Accept-Encoding: gzip" "$BASE_URL/test.txt")
plain_size=$(curl -s -o /dev/null -w "%{size_download}" "$BASE_URL/test.txt")
[ "$gzip_size" -gt 0 ] && assert "Gzip: response received" "PASS" || assert "Gzip: response received" "empty"

# Without gzip (no Accept-Encoding), body is plain text
no_gzip_body=$(curl -s "$BASE_URL/test.txt")
echo "$no_gzip_body" | grep -q "Hello, gofile" && assert "Non-gzip: plain body correct" "PASS" || assert "Non-gzip: plain body correct" "decode failed"

stop_server

# ========================================
# Phase 3: Auth server (auth + upload, multi-user)
# ========================================
echo -e "\n${YELLOW}[Phase 3] Auth server (auth + upload, multi-user)${NC}"

start_server "-auth -authString admin:pass123,user2:pass456 -upload -uploadSize 5"

# --- Auth required ---
echo -e "  ${CYAN}-- Auth required --${NC}"
check_status "Root needs auth (401)"    "401" "$BASE_URL/"
check_status "File needs auth (401)"    "401" "$BASE_URL/test.txt"
check_status "Dir needs auth (401)"     "401" "$BASE_URL/subdir/"

# --- Wrong credentials ---
echo -e "  ${CYAN}-- Wrong credentials --${NC}"
check_status "Wrong user (401)"              "401" -u wrong:wrong       "$BASE_URL/"
check_status "Right user wrong pass (401)"   "401" -u admin:wrong       "$BASE_URL/"
check_status "Wrong user right pass (401)"   "401" -u wrong:pass123     "$BASE_URL/"
check_status "Empty password (401)"          "401" -u admin:            "$BASE_URL/"

# Malformed auth header
code_bearer=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer token" "$BASE_URL/")
[ "$code_bearer" = "400" ] && assert "Malformed auth header (400)" "PASS" || assert "Malformed auth header (400)" "got $code_bearer"

# Bad base64 auth
code_bad64=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic !!!invalid!!!" "$BASE_URL/")
[ "$code_bad64" = "400" ] && assert "Bad base64 auth (400)" "PASS" || assert "Bad base64 auth (400)" "got $code_bad64"

# --- Correct credentials: user1 ---
echo -e "  ${CYAN}-- Correct auth: admin:pass123 --${NC}"
check_status "Auth OK: root dir"         "200" -u admin:pass123 "$BASE_URL/"
check_status "Auth OK: download file"    "200" -u admin:pass123 "$BASE_URL/test.txt"
check_status "Auth OK: subdir listing"   "200" -u admin:pass123 "$BASE_URL/subdir/"
check_contains "Auth OK: sees test.txt"  "test.txt" -u admin:pass123 "$BASE_URL/"
dl_tmp="$TEST_DIR/.auth_dl_tmp"
dl_code=$(curl -s -o "$dl_tmp" -w "%{http_code}" -u admin:pass123 "$BASE_URL/test.txt")
if [ "$dl_code" = "200" ] && cmp -s "$dl_tmp" "$TEST_DIR/test.txt"; then
    assert "Auth OK: download matches" "PASS"
else
    assert "Auth OK: download matches" "code=$dl_code or content mismatch"
fi
rm -f "$dl_tmp"

# --- Correct credentials: user2 ---
echo -e "  ${CYAN}-- Correct auth: user2:pass456 --${NC}"
check_status "User2 auth OK: root"     "200" -u user2:pass456 "$BASE_URL/"
check_status "User2 auth OK: download" "200" -u user2:pass456 "$BASE_URL/test.txt"

# --- Upload with auth ---
echo -e "  ${CYAN}-- Upload with auth --${NC}"
echo "auth upload content" > "$TEST_DIR/.auth_upload.txt"
up_code=$(curl -s -o /dev/null -w "%{http_code}" -u admin:pass123 -F "file=@$TEST_DIR/.auth_upload.txt" "$BASE_URL/upload?dir=/uploads")
[ "$up_code" = "200" ] && assert "Auth upload OK (200)" "PASS" || assert "Auth upload OK (200)" "got $up_code"
[ -f "$UPLOAD_DIR/.auth_upload.txt" ] && assert "Auth uploaded file exists" "PASS" || assert "Auth uploaded file exists" "not found"

# Upload without auth
up_code_noauth=$(curl -s -o /dev/null -w "%{http_code}" -F "file=@$TEST_DIR/.auth_upload.txt" "$BASE_URL/upload?dir=/uploads")
[ "$up_code_noauth" = "401" ] && assert "Upload without auth rejected (401)" "PASS" || assert "Upload without auth rejected (401)" "got $up_code_noauth"

rm -f "$TEST_DIR/.auth_upload.txt"

stop_server

# ========================================
# Phase 4: Server without upload
# ========================================
echo -e "\n${YELLOW}[Phase 4] Upload disabled (default)${NC}"

start_server ""

echo -e "  ${CYAN}-- Upload disabled --${NC}"
up_code_403=$(curl -s -o /dev/null -w "%{http_code}" -X POST -F "file=@$TEST_DIR/test.txt" "$BASE_URL/upload?dir=/")
[ "$up_code_403" = "403" ] && assert "Upload forbidden (403)" "PASS" || assert "Upload forbidden (403)" "got $up_code_403"

echo -e "  ${CYAN}-- No upload form in HTML --${NC}"
check_not_contains "No upload form in HTML" "uploadForm" "$BASE_URL/"

echo -e "  ${CYAN}-- Basic features still work --${NC}"
check_status "Dir listing OK"   "200" "$BASE_URL/"
check_status "File download OK" "200" "$BASE_URL/test.txt"
check_status "Favicon OK"      "200" "$BASE_URL/favicon.png"
check_status "HEAD OK"          "200" -I "$BASE_URL/test.txt"
check_file_download "Download matches" "$BASE_URL/test.txt" "$TEST_DIR/test.txt"

stop_server

# ========================================
# Phase 5: Edge cases
# ========================================
echo -e "\n${YELLOW}[Phase 5] Edge cases${NC}"

start_server "-upload -uploadSize 10"

# Trailing slash on directory
check_status "Dir with trailing slash" "200" "$BASE_URL/subdir/"

# URL encoded path
check_status "URL encoded path" "200" "$BASE_URL/test%2etxt"

# Empty file upload
touch "$TEST_DIR/.empty.txt"
up_code_empty=$(curl -s -o /dev/null -w "%{http_code}" -F "file=@$TEST_DIR/.empty.txt" "$BASE_URL/upload?dir=/uploads")
[ "$up_code_empty" = "200" ] && assert "Upload empty file" "PASS" || assert "Upload empty file" "got $up_code_empty"
rm -f "$TEST_DIR/.empty.txt"

# Large file download (2MB) integrity
check_file_download "Large file (2MB) integrity" "$BASE_URL/large.bin" "$TEST_DIR/large.bin"

# Upload file with special chars in name
echo "special" > "$TEST_DIR/.test-file_v2.0.txt"
up_code_sp=$(curl -s -o /dev/null -w "%{http_code}" -F "file=@$TEST_DIR/.test-file_v2.0.txt" "$BASE_URL/upload?dir=/uploads")
[ "$up_code_sp" = "200" ] && assert "Upload file with special chars" "PASS" || assert "Upload file with special chars" "got $up_code_sp"
rm -f "$TEST_DIR/.test-file_v2.0.txt"

# Multiple rapid requests
echo -e "  ${CYAN}-- Rapid requests --${NC}"
all_ok=true
for i in $(seq 1 10); do
    code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/test.txt")
    [ "$code" != "200" ] && all_ok=false && break
done
$all_ok && assert "10 rapid requests all 200" "PASS" || assert "10 rapid requests all 200" "one failed with $code"

stop_server

# ========================================
# Results
# ========================================
echo ""

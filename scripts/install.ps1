#
# gofile installer for Windows
# Usage: irm https://raw.githubusercontent.com/mars-base/gofile/main/install.ps1 | iex
#   or: iwr -useb https://raw.githubusercontent.com/mars-base/gofile/main/install.ps1 -outfile install.ps1; .\install.ps1 -Version "v1.0.0"
#   or: .\install.ps1 -InstallDir "C:\bin"
#
param(
    [string]$Version = "",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

$Repo = "mars-base/gofile"
$BinaryName = "gofile.exe"

# --- Default install directory ---
if ([string]::IsNullOrEmpty($InstallDir)) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "gofile"
}

# --- Colors ---
function Write-Info  { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Green }
function Write-Warn  { param($msg) Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Write-Err   { param($msg) Write-Host "[ERROR] $msg" -ForegroundColor Red }

# --- Get latest version from GitHub ---
function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $headers = @{ "User-Agent" = "gofile-installer" }
        $release = Invoke-RestMethod -Uri $apiUrl -Headers $headers -UseBasicParsing
        return $release.tag_name
    } catch {
        Write-Err "Failed to fetch latest version from GitHub: $_"
        exit 1
    }
}

# --- Check dependencies ---
function Test-Dependencies {
    # PowerShell 5.1+ has Invoke-WebRequest built-in
    if ($PSVersionTable.PSVersion.Major -lt 5) {
        Write-Err "PowerShell 5.1+ required (current: $($PSVersionTable.PSVersion))"
        exit 1
    }
}

# --- Main ---
function Install-Gofile {
    Write-Host ""
    Write-Host "gofile installer for Windows" -ForegroundColor Cyan
    Write-Host ""

    Test-Dependencies

    # Get version
    if ([string]::IsNullOrEmpty($Version)) {
        Write-Info "Fetching latest version..."
        $Version = Get-LatestVersion
    }
    Write-Info "Version: $Version"
    Write-Info "Install directory: $InstallDir"

    # Determine download URL
    $archiveName = "gofile.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"
    Write-Info "Downloading: $downloadUrl"

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        Write-Info "Creating directory: $InstallDir"
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # Download
    $tmpFile = Join-Path $env:TEMP "gofile-$Version.zip"
    try {
        $ProgressPreference = 'SilentlyContinue'  # Speed up download
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tmpFile -UseBasicParsing
        $ProgressPreference = 'Continue'
    } catch {
        Write-Err "Download failed: $downloadUrl"
        Write-Err "Error: $_"
        exit 1
    }

    # Extract
    Write-Info "Extracting..."
    try {
        # Remove old binary if exists
        $binaryPath = Join-Path $InstallDir $BinaryName
        if (Test-Path $binaryPath) {
            Remove-Item $binaryPath -Force
        }

        Expand-Archive -Path $tmpFile -DestinationPath $InstallDir -Force
    } catch {
        Write-Err "Extraction failed: $_"
        Remove-Item $tmpFile -Force -ErrorAction SilentlyContinue
        exit 1
    }

    # Cleanup
    Remove-Item $tmpFile -Force -ErrorAction SilentlyContinue

    # Verify
    $binaryPath = Join-Path $InstallDir $BinaryName
    if (Test-Path $binaryPath) {
        Write-Host ""
        Write-Info "Installation successful!"
        Write-Info "  Binary: $binaryPath"

        # Show version
        try {
            $verOutput = & $binaryPath -v 2>&1
            Write-Info "  $verOutput"
        } catch {
            Write-Warn "Could not verify installation"
        }

        Write-Host ""

        # Check if install dir is in PATH
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($currentPath -notlike "*$InstallDir*") {
            Write-Warn "$InstallDir is not in your PATH"
            Write-Info "Adding to user PATH..."

            $newPath = if ([string]::IsNullOrEmpty($currentPath)) {
                $InstallDir
            } else {
                "$currentPath;$InstallDir"
            }
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")

            # Update current session PATH
            $env:PATH = "$env:PATH;$InstallDir"

            Write-Info "Added to PATH (restart terminal to use 'gofile' command)"
        } else {
            Write-Info "Run 'gofile' to start the server"
        }
    } else {
        Write-Err "Installation failed: binary not found at $binaryPath"
        exit 1
    }
}

Install-Gofile

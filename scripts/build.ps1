# -----------------------------------------------------------------------
# Build Script for Aktis Collector Jira
# -----------------------------------------------------------------------

param (
    [string]$Environment = "dev",
    [string]$Version = "",
    [switch]$Clean,
    [switch]$Test,
    [switch]$Verbose,
    [switch]$Release,
    [switch]$Docker
)

<#
.SYNOPSIS
    Build script for Aktis Collector Jira

.DESCRIPTION
    This script builds the Aktis Collector Jira for local development and testing.
    Outputs the executable to the project's bin directory.

.PARAMETER Environment
    Target environment for build (dev, staging, prod)

.PARAMETER Version
    Version to embed in the binary (defaults to .version file or git commit hash)

.PARAMETER Clean
    Clean build artifacts before building

.PARAMETER Test
    Run tests before building

.PARAMETER Verbose
    Enable verbose output

.PARAMETER Release
    Build optimized release binary

.PARAMETER Docker
    Build for Docker deployment (skip stopping local process)

.EXAMPLE
    .\build.ps1
    Build aktis collector for development

.EXAMPLE
    .\build.ps1 -Release
    Build optimized release version

.EXAMPLE
    .\build.ps1 -Environment prod -Version "1.0.0"
    Build for production with specific version
#>

# Error handling
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Build configuration
$gitCommit = ""

try {
    $gitCommit = git rev-parse --short HEAD 2>$null
    if (-not $gitCommit) { $gitCommit = "unknown" }
}
catch {
    $gitCommit = "unknown"
}

Write-Host "Aktis Collector Jira Build Script" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan

# Setup paths
$scriptDir = $PSScriptRoot
$projectRoot = Split-Path -Parent $scriptDir
$versionFilePath = Join-Path -Path $projectRoot -ChildPath ".version"
$binDir = Join-Path -Path $projectRoot -ChildPath "bin"
$collectorOutputPath = Join-Path -Path $binDir -ChildPath "aktis-collector-jira.exe"

Write-Host "Project Root: $projectRoot" -ForegroundColor Gray
Write-Host "Environment: $Environment" -ForegroundColor Gray
Write-Host "Git Commit: $gitCommit" -ForegroundColor Gray

# Handle version file creation and maintenance
$buildTimestamp = Get-Date -Format "MM-dd-HH-mm-ss"

if (-not (Test-Path $versionFilePath)) {
    # Create .version file if it doesn't exist
    $versionContent = @"
version: 0.1.0
build: $buildTimestamp
"@
    Set-Content -Path $versionFilePath -Value $versionContent
    Write-Host "Created .version file with version 0.1.0" -ForegroundColor Green
} else {
    # Read current version and increment patch version
    $versionLines = Get-Content $versionFilePath
    $currentVersion = ""
    $updatedLines = @()

    foreach ($line in $versionLines) {
        if ($line -match '^version:\s*(.+)$') {
            $currentVersion = $matches[1].Trim()

            # Parse version (format: major.minor.patch)
            if ($currentVersion -match '^(\d+)\.(\d+)\.(\d+)$') {
                $major = [int]$matches[1]
                $minor = [int]$matches[2]
                $patch = [int]$matches[3]

                # Increment patch version
                $patch++
                $newVersion = "$major.$minor.$patch"

                $updatedLines += "version: $newVersion"
                Write-Host "Incremented version: $currentVersion -> $newVersion" -ForegroundColor Green
            } else {
                # Version format not recognized, keep as-is
                $updatedLines += $line
                Write-Host "Version format not recognized, keeping: $currentVersion" -ForegroundColor Yellow
            }
        } elseif ($line -match '^build:\s*') {
            $updatedLines += "build: $buildTimestamp"
        } else {
            $updatedLines += $line
        }
    }

    Set-Content -Path $versionFilePath -Value $updatedLines
    Write-Host "Updated build timestamp to: $buildTimestamp" -ForegroundColor Green
}

# Read version information from .version file
$versionInfo = @{}
$versionLines = Get-Content $versionFilePath
foreach ($line in $versionLines) {
    if ($line -match '^version:\s*(.+)$') {
        $versionInfo.Version = $matches[1].Trim()
    }
    if ($line -match '^build:\s*(.+)$') {
        $versionInfo.Build = $matches[1].Trim()
    }
}

Write-Host "Using version: $($versionInfo.Version), build: $($versionInfo.Build)" -ForegroundColor Cyan

# Clean build artifacts if requested
if ($Clean) {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Yellow
    if (Test-Path $binDir) {
        Remove-Item -Path $binDir -Recurse -Force
    }
    if (Test-Path "go.sum") {
        Remove-Item -Path "go.sum" -Force
    }
}

# Create bin directory
if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir | Out-Null
}

# Run tests if requested
if ($Test) {
    Write-Host "Running tests..." -ForegroundColor Yellow
    go test ./... -v
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Tests failed!" -ForegroundColor Red
        exit 1
    }
    Write-Host "Tests passed!" -ForegroundColor Green
}

# Stop executing process if it's running (skip for Docker builds)
if (-not $Docker) {
    try {
        $processName = "aktis-collector-jira"
        $process = Get-Process -Name $processName -ErrorAction SilentlyContinue

        if ($process) {
            Write-Host "Stopping existing Aktis Collector Jira process..." -ForegroundColor Yellow
            Stop-Process -Name $processName -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 1  # Give process time to fully terminate
            Write-Host "Process stopped successfully" -ForegroundColor Green
        } else {
            Write-Host "No Aktis Collector Jira process found running" -ForegroundColor Gray
        }
    }
    catch {
        Write-Warning "Could not stop Aktis Collector Jira process: $($_.Exception.Message)"
    }
} else {
    Write-Host "Docker build - skipping local process check" -ForegroundColor Cyan
}

# Download dependencies
Write-Host "Downloading dependencies..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to download dependencies!" -ForegroundColor Red
    exit 1
}

# Build flags
$module = "aktis-collector-jira/internal/common"
$buildFlags = @(
    "-X", "$module.Version=$($versionInfo.Version)",
    "-X", "$module.Build=$($versionInfo.Build)",
    "-X", "$module.GitCommit=$gitCommit"
)

if ($Release) {
    $buildFlags += @("-w", "-s")  # Strip debug info and symbol table
}

$ldflags = $buildFlags -join " "

# Build command
Write-Host "Building aktis-collector-jira..." -ForegroundColor Yellow

$env:CGO_ENABLED = "0"
if ($Release) {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
}

$buildArgs = @(
    "build"
    "-ldflags=$ldflags"
    "-o", $collectorOutputPath
    ".\cmd\aktis-collector-jira"
)

# Change to project root for build
Push-Location $projectRoot

if ($Verbose) {
    $buildArgs += "-v"
}

Write-Host "Build command: go $($buildArgs -join ' ')" -ForegroundColor Gray

& go @buildArgs

# Return to original directory
Pop-Location

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Copy configuration file to bin directory
$configSourcePath = Join-Path -Path $projectRoot -ChildPath "deployments\aktis-collector-jira.toml"
$configDestPath = Join-Path -Path $binDir -ChildPath "aktis-collector-jira.toml"

if (Test-Path $configSourcePath) {
    if (-not (Test-Path $configDestPath)) {
        Copy-Item -Path $configSourcePath -Destination $configDestPath
        Write-Host "Copied configuration: deployments/aktis-collector-jira.toml -> bin/" -ForegroundColor Green
    } else {
        Write-Host "Using existing bin/aktis-collector-jira.toml (preserving customizations)" -ForegroundColor Cyan
    }
}

# Copy pages directory for web interface
$pagesSourcePath = Join-Path -Path $projectRoot -ChildPath "pages"
$pagesDestPath = Join-Path -Path $binDir -ChildPath "pages"

if (Test-Path $pagesSourcePath) {
    if (Test-Path $pagesDestPath) {
        Remove-Item -Path $pagesDestPath -Recurse -Force
    }
    Copy-Item -Path $pagesSourcePath -Destination $pagesDestPath -Recurse
    Write-Host "Copied web interface: pages/ -> bin/pages/" -ForegroundColor Green
}

# Build and deploy Chrome Extension
Write-Host "`nBuilding Chrome Extension..." -ForegroundColor Yellow

$extensionSourcePath = Join-Path -Path $projectRoot -ChildPath "cmd\aktis-chrome-extension"
$extensionDestPath = Join-Path -Path $binDir -ChildPath "aktis-chrome-extension"

# Check if extension source exists
if (Test-Path $extensionSourcePath) {
    # Create extension directory in bin
    if (Test-Path $extensionDestPath) {
        Remove-Item -Path $extensionDestPath -Recurse -Force
    }
    New-Item -ItemType Directory -Path $extensionDestPath | Out-Null

    # Copy extension files (exclude create-icons.ps1 as it's a dev tool)
    $extensionFiles = @(
        "manifest.json",
        "background.js",
        "content.js",
        "popup.html",
        "popup.js",
        "README.md"
    )

    foreach ($file in $extensionFiles) {
        $sourcePath = Join-Path -Path $extensionSourcePath -ChildPath $file
        if (Test-Path $sourcePath) {
            Copy-Item -Path $sourcePath -Destination $extensionDestPath
        } else {
            Write-Warning "Extension file not found: $file"
        }
    }

    # Copy icons directory
    $iconsSourcePath = Join-Path -Path $extensionSourcePath -ChildPath "icons"
    $iconsDestPath = Join-Path -Path $extensionDestPath -ChildPath "icons"

    if (Test-Path $iconsSourcePath) {
        Copy-Item -Path $iconsSourcePath -Destination $iconsDestPath -Recurse
        Write-Host "Copied extension icons: icons/ -> bin/aktis-chrome-extension/icons/" -ForegroundColor Green
    } else {
        # Icons don't exist, create them
        Write-Host "Icons not found, creating placeholder icons..." -ForegroundColor Yellow

        New-Item -ItemType Directory -Path $iconsDestPath -Force | Out-Null

        $createIconScript = Join-Path -Path $extensionSourcePath -ChildPath "create-icons.ps1"
        if (Test-Path $createIconScript) {
            & powershell.exe -ExecutionPolicy Bypass -File $createIconScript
            # Copy newly created icons
            if (Test-Path $iconsSourcePath) {
                Copy-Item -Path $iconsSourcePath -Destination $iconsDestPath -Recurse -Force
                Write-Host "Created and copied extension icons" -ForegroundColor Green
            }
        } else {
            Write-Warning "Icon creation script not found, extension may not have icons"
        }
    }

    Write-Host "Deployed Chrome Extension: bin/aktis-chrome-extension/" -ForegroundColor Green
} else {
    Write-Warning "Chrome extension source not found at: $extensionSourcePath"
}

# Verify executable was created
if (-not (Test-Path $collectorOutputPath)) {
    Write-Error "Build completed but executable not found: $collectorOutputPath"
    exit 1
}

# Get file info for binary
$fileInfo = Get-Item $collectorOutputPath
$fileSizeMB = [math]::Round($fileInfo.Length / 1MB, 2)

Write-Host "`n==== Build Summary ====" -ForegroundColor Cyan
Write-Host "Status: SUCCESS" -ForegroundColor Green
Write-Host "Environment: $Environment" -ForegroundColor Green
Write-Host "Version: $($versionInfo.Version)" -ForegroundColor Green
Write-Host "Build: $($versionInfo.Build)" -ForegroundColor Green
Write-Host "Collector Output: $collectorOutputPath ($fileSizeMB MB)" -ForegroundColor Green
Write-Host "Extension Output: $extensionDestPath" -ForegroundColor Green
Write-Host "Build Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Green

if ($Test) {
    Write-Host "Tests: EXECUTED" -ForegroundColor Green
}

if ($Clean) {
    Write-Host "Clean: EXECUTED" -ForegroundColor Green
}

Write-Host "`nBuild completed successfully!" -ForegroundColor Green
Write-Host "Server: $collectorOutputPath" -ForegroundColor Cyan
Write-Host "Extension: $extensionDestPath" -ForegroundColor Cyan
Write-Host "`nTo install extension:" -ForegroundColor Yellow
Write-Host "  1. Open Chrome and go to chrome://extensions/" -ForegroundColor Gray
Write-Host "  2. Enable 'Developer mode'" -ForegroundColor Gray
Write-Host "  3. Click 'Load unpacked'" -ForegroundColor Gray
Write-Host "  4. Select: $extensionDestPath" -ForegroundColor Gray
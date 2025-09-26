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
$buildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
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
$projectRoot = Get-Location
$versionFilePath = Join-Path -Path $projectRoot -ChildPath ".version"
$binDir = Join-Path -Path $projectRoot -ChildPath "bin"
$collectorOutputPath = Join-Path -Path $binDir -ChildPath "aktis-collector-jira.exe"

Write-Host "Project Root: $projectRoot" -ForegroundColor Gray
Write-Host "Environment: $Environment" -ForegroundColor Gray
Write-Host "Git Commit: $gitCommit" -ForegroundColor Gray
Write-Host "Build Time: $buildTime" -ForegroundColor Gray

# Handle version management
if ([string]::IsNullOrWhiteSpace($Version)) {
    # Try to read version from .version file
    if (Test-Path $versionFilePath) {
        $versionContent = Get-Content $versionFilePath | Where-Object { $_ -match '\S' } | Select-Object -First 1
        $Version = $versionContent.Trim()
        Write-Host "Using version from .version file: $Version" -ForegroundColor Green
    }
    else {
        # Default version
        $Version = "1.0.0-dev"
        Write-Host "Using default version: $Version" -ForegroundColor Yellow

        # Create .version file
        Set-Content -Path $versionFilePath -Value $Version
    }
}
else {
    Write-Host "Using specified version: $Version" -ForegroundColor Green
    # Update .version file
    Set-Content -Path $versionFilePath -Value $Version
}

Write-Host "Final Version: $Version" -ForegroundColor Green

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
    "-X", "$module.Version=$Version",
    "-X", "$module.Build=$buildTime",
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

if ($Verbose) {
    $buildArgs += "-v"
}

Write-Host "Build command: go $($buildArgs -join ' ')" -ForegroundColor Gray

& go @buildArgs

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Success message
Write-Host "" -ForegroundColor Green
Write-Host "Build completed successfully!" -ForegroundColor Green
Write-Host "Executable: $collectorOutputPath" -ForegroundColor Green

# Show binary info
if (Test-Path $collectorOutputPath) {
    $fileInfo = Get-Item $collectorOutputPath
    Write-Host "Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Gray
    Write-Host "Created: $($fileInfo.CreationTime)" -ForegroundColor Gray
}

Write-Host "" -ForegroundColor Green
Write-Host "Usage examples:" -ForegroundColor Cyan
Write-Host "  $collectorOutputPath -help" -ForegroundColor White
Write-Host "  $collectorOutputPath -version" -ForegroundColor White
Write-Host "  $collectorOutputPath -config config.json" -ForegroundColor White
Write-Host "  $collectorOutputPath -config config.json -update" -ForegroundColor White
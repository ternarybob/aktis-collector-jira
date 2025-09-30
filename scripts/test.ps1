# -----------------------------------------------------------------------
# Test Script for Aktis Collector Jira
# -----------------------------------------------------------------------

param (
    [switch]$Unit,
    [switch]$Integration,
    [switch]$Coverage,
    [switch]$Verbose,
    [switch]$Race,
    [switch]$Short,
    [string]$Package = "./...",
    [string]$Run = ""
)

<#
.SYNOPSIS
    Run tests for Aktis Collector Jira

.DESCRIPTION
    This script runs unit tests, integration tests, and generates coverage reports
    for the Aktis Collector Jira project.

.PARAMETER Unit
    Run unit tests only

.PARAMETER Integration
    Run integration tests only

.PARAMETER Coverage
    Generate coverage report

.PARAMETER Verbose
    Enable verbose test output

.PARAMETER Race
    Enable race detector

.PARAMETER Short
    Run tests in short mode

.PARAMETER Package
    Specific package to test (default: ./...)

.PARAMETER Run
    Run only tests matching the pattern

.EXAMPLE
    .\test.ps1
    Run all tests

.EXAMPLE
    .\test.ps1 -Coverage
    Run tests with coverage report

.EXAMPLE
    .\test.ps1 -Unit -Verbose
    Run unit tests with verbose output

.EXAMPLE
    .\test.ps1 -Package "./internal/services" -Run "TestStorage"
    Run specific test in specific package
#>

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Color output
function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

# Setup paths
$scriptDir = $PSScriptRoot
$projectRoot = Split-Path -Parent $scriptDir
$coverageDir = Join-Path -Path $projectRoot -ChildPath "coverage"
$coverageFile = Join-Path -Path $coverageDir -ChildPath "coverage.out"
$coverageHtml = Join-Path -Path $coverageDir -ChildPath "coverage.html"

Write-ColorOutput "Aktis Collector Jira Test Script" "Cyan"
Write-ColorOutput "=================================" "Cyan"

# Ensure we're in project root
Push-Location $projectRoot

try {
    # Build test arguments
    $testArgs = @("test")

    # Add package
    $testArgs += $Package

    # Add verbose flag
    if ($Verbose) {
        $testArgs += "-v"
    }

    # Add race detector
    if ($Race) {
        $testArgs += "-race"
    }

    # Add short mode
    if ($Short) {
        $testArgs += "-short"
    }

    # Add run pattern
    if ($Run) {
        $testArgs += "-run"
        $testArgs += $Run
    }

    # Add build tags based on test type
    if ($Unit) {
        Write-ColorOutput "`nRunning UNIT tests..." "Yellow"
        $testArgs += "-tags=unit"
    } elseif ($Integration) {
        Write-ColorOutput "`nRunning INTEGRATION tests..." "Yellow"
        $testArgs += "-tags=integration"
    } else {
        Write-ColorOutput "`nRunning ALL tests..." "Yellow"
    }

    # Add coverage
    if ($Coverage) {
        Write-ColorOutput "Coverage reporting enabled" "Gray"

        # Create coverage directory
        if (-not (Test-Path $coverageDir)) {
            New-Item -ItemType Directory -Path $coverageDir | Out-Null
        }

        $testArgs += "-coverprofile=$coverageFile"
        $testArgs += "-covermode=atomic"
    }

    Write-ColorOutput "Test command: go $($testArgs -join ' ')" "Gray"
    Write-ColorOutput ""

    # Run tests
    $startTime = Get-Date

    & go @testArgs
    $testExitCode = $LASTEXITCODE

    $endTime = Get-Date
    $duration = $endTime - $startTime

    Write-ColorOutput ""
    Write-ColorOutput "==== Test Summary ====" "Cyan"
    Write-ColorOutput "Duration: $($duration.TotalSeconds) seconds" "Gray"

    if ($testExitCode -eq 0) {
        Write-ColorOutput "Status: PASSED" "Green"

        # Generate coverage report
        if ($Coverage -and (Test-Path $coverageFile)) {
            Write-ColorOutput "`nGenerating coverage report..." "Yellow"

            # Get coverage percentage
            $coverageOutput = & go tool cover -func=$coverageFile
            $totalLine = $coverageOutput | Select-String "total:"

            if ($totalLine) {
                $coveragePercent = $totalLine.ToString() -replace '.*total:\s+\(statements\)\s+(\d+\.\d+)%.*', '$1'
                Write-ColorOutput "Coverage: $coveragePercent%" "Cyan"
            }

            # Generate HTML report
            & go tool cover -html=$coverageFile -o $coverageHtml
            Write-ColorOutput "HTML Report: $coverageHtml" "Cyan"

            # Open in browser (optional)
            # Start-Process $coverageHtml
        }

    } else {
        Write-ColorOutput "Status: FAILED" "Red"
    }

    exit $testExitCode

} finally {
    Pop-Location
}

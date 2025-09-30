# -----------------------------------------------------------------------
# Favicon Generator Script for Aktis Collector Jira
# -----------------------------------------------------------------------

<#
.SYNOPSIS
    Generate favicon files for the web interface

.DESCRIPTION
    Creates a simple favicon from text or generates placeholder favicons
    for the web interface. Requires PowerShell 5.1+ on Windows.

.EXAMPLE
    .\create-favicon.ps1
    Generate default favicon with "AJ" text
#>

param (
    [string]$Text = "AJ",
    [string]$OutputDir = "..\pages",
    [string]$BackgroundColor = "#2563eb",
    [string]$ForegroundColor = "#ffffff"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Color output
function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

Write-ColorOutput "Aktis Collector Jira Favicon Generator" "Cyan"
Write-ColorOutput "=======================================" "Cyan"

# Setup paths
$scriptDir = $PSScriptRoot
$projectRoot = Split-Path -Parent $scriptDir

if (-not [System.IO.Path]::IsPathRooted($OutputDir)) {
    $OutputDir = Join-Path -Path $scriptDir -ChildPath $OutputDir
}

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

$faviconPath = Join-Path -Path $OutputDir -ChildPath "favicon.ico"
$pngPath = Join-Path -Path $OutputDir -ChildPath "favicon.png"

Write-ColorOutput "Output directory: $OutputDir" "Gray"
Write-ColorOutput "Text: $Text" "Gray"
Write-ColorOutput "Background: $BackgroundColor" "Gray"
Write-ColorOutput "Foreground: $ForegroundColor" "Gray"

# Check if System.Drawing is available (Windows PowerShell)
$canGenerateImage = $false
try {
    Add-Type -AssemblyName System.Drawing
    $canGenerateImage = $true
} catch {
    Write-ColorOutput "System.Drawing not available" "Yellow"
}

if ($canGenerateImage) {
    Write-ColorOutput "`nGenerating favicon..." "Yellow"

    # Create bitmap
    $size = 64
    $bitmap = New-Object System.Drawing.Bitmap($size, $size)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)

    # Set high quality rendering
    $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias

    # Parse colors
    $bgColor = [System.Drawing.ColorTranslator]::FromHtml($BackgroundColor)
    $fgColor = [System.Drawing.ColorTranslator]::FromHtml($ForegroundColor)

    # Draw background
    $brush = New-Object System.Drawing.SolidBrush($bgColor)
    $graphics.FillRectangle($brush, 0, 0, $size, $size)

    # Draw text
    $font = New-Object System.Drawing.Font("Arial", 28, [System.Drawing.FontStyle]::Bold)
    $textBrush = New-Object System.Drawing.SolidBrush($fgColor)
    $format = New-Object System.Drawing.StringFormat
    $format.Alignment = [System.Drawing.StringAlignment]::Center
    $format.LineAlignment = [System.Drawing.StringAlignment]::Center

    $rect = New-Object System.Drawing.RectangleF(0, 0, $size, $size)
    $graphics.DrawString($Text, $font, $textBrush, $rect, $format)

    # Save as PNG
    $bitmap.Save($pngPath, [System.Drawing.Imaging.ImageFormat]::Png)

    # Clean up
    $graphics.Dispose()
    $bitmap.Dispose()
    $brush.Dispose()
    $textBrush.Dispose()
    $font.Dispose()

    Write-ColorOutput "PNG favicon created: $pngPath" "Green"

    # Note: Converting PNG to ICO requires additional tools
    # For now, we'll just use the PNG file
    Write-ColorOutput "Note: Using PNG favicon. For ICO format, use an online converter." "Yellow"

} else {
    # Fallback: Create SVG favicon
    Write-ColorOutput "`nGenerating SVG favicon (fallback)..." "Yellow"

    $svgPath = Join-Path -Path $OutputDir -ChildPath "favicon.svg"

    $svgContent = @"
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <rect width="64" height="64" fill="$BackgroundColor"/>
  <text x="32" y="40" font-family="Arial, sans-serif" font-size="28" font-weight="bold"
        text-anchor="middle" fill="$ForegroundColor">$Text</text>
</svg>
"@

    Set-Content -Path $svgPath -Value $svgContent -Encoding UTF8
    Write-ColorOutput "SVG favicon created: $svgPath" "Green"
    Write-ColorOutput "Add this to your HTML: <link rel='icon' type='image/svg+xml' href='favicon.svg'>" "Cyan"
}

Write-ColorOutput "`nFavicon generation completed!" "Green"

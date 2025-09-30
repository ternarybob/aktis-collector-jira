# Create placeholder icons for Chrome extension

Add-Type -AssemblyName System.Drawing

$sizes = @(16, 48, 128)
$iconDir = Join-Path $PSScriptRoot "icons"

foreach ($size in $sizes) {
    $bmp = New-Object System.Drawing.Bitmap($size, $size)
    $g = [System.Drawing.Graphics]::FromImage($bmp)

    # Green background
    $g.Clear([System.Drawing.Color]::FromArgb(0, 204, 0))

    # White "A" text
    $fontSize = [Math]::Floor($size * 0.4)
    $font = New-Object System.Drawing.Font('Arial', $fontSize, [System.Drawing.FontStyle]::Bold)
    $brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)

    $text = 'A'
    $sf = New-Object System.Drawing.StringFormat
    $sf.Alignment = [System.Drawing.StringAlignment]::Center
    $sf.LineAlignment = [System.Drawing.StringAlignment]::Center

    $rect = New-Object System.Drawing.RectangleF(0, 0, $size, $size)
    $g.DrawString($text, $font, $brush, $rect, $sf)

    # Save PNG
    $iconPath = Join-Path $iconDir "icon$size.png"
    $bmp.Save($iconPath, [System.Drawing.Imaging.ImageFormat]::Png)

    Write-Host "Created $iconPath"

    # Cleanup
    $g.Dispose()
    $bmp.Dispose()
}

Write-Host "Icon creation complete"
#!/bin/bash

# Jira Collector Build Script

echo "🔧 Building Jira Collector Plugin..."

# Clean previous builds
rm -f jira-collector
rm -rf data/
rm -rf backups/

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download

# Build the plugin
echo "🏗️  Building plugin..."
go build -o jira-collector main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "📁 Creating data directories..."
    mkdir -p data backups
    echo "✅ Directories created"
    echo ""
    echo "🎯 Plugin ready to use!"
    echo ""
    echo "Usage examples:"
    echo "  ./jira-collector -help"
    echo "  ./jira-collector -config config.example.json"
    echo "  ./jira-collector -mode prod -quiet"
    echo "  ./jira-collector -update"
else
    echo "❌ Build failed!"
    exit 1
fi
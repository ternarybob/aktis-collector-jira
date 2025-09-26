#!/bin/bash

# Jira Collector Deployment Script

echo "🚀 Deploying Jira Collector System..."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.24 or higher."
    exit 1
fi

print_status "Building Jira Collector Plugin..."
cd jira-collector

# Clean previous builds
rm -f jira-collector
rm -rf data/
rm -rf backups/

# Download dependencies
print_status "Downloading dependencies..."
go mod download
if [ $? -ne 0 ]; then
    print_error "Failed to download dependencies"
    exit 1
fi

# Build the plugin
print_status "Building plugin..."
go build -o jira-collector main.go
if [ $? -ne 0 ]; then
    print_error "Failed to build plugin"
    exit 1
fi

# Create data directories
mkdir -p data backups
print_success "Plugin built successfully!"

cd ..

print_status "Setting up Web Interface..."
cd web-interface

# Check if we need to build the web server
if [ ! -f server.go ]; then
    print_error "Web server not found"
    exit 1
fi

# Build the web server
print_status "Building web server..."
go build -o dashboard-server server.go
if [ $? -ne 0 ]; then
    print_error "Failed to build web server"
    exit 1
fi

print_success "Web interface ready!"

cd ..

print_status "Creating configuration files..."

# Create a simple startup script
cat > start-system.sh << 'EOF'
#!/bin/bash

echo "🚀 Starting Jira Collector System..."

# Start the web interface
echo "🌐 Starting Dashboard..."
cd web-interface
./dashboard-server &
DASHBOARD_PID=$!
echo "Dashboard PID: $DASHBOARD_PID"

cd ..

# Start the collector (in development mode for testing)
echo "🎯 Starting Jira Collector..."
cd jira-collector
echo "You can now run: ./jira-collector -help"
echo ""
echo "📊 Dashboard available at: http://localhost:8080"
echo ""
echo "To start collecting data:"
echo "  ./jira-collector -config config.example.json"
echo ""
echo "Press Ctrl+C to stop the system"

# Wait for dashboard
wait $DASHBOARD_PID
EOF

chmod +x start-system.sh

print_success "Deployment complete!"
echo ""
echo "📁 System Structure:"
echo "  jira-collector/          # Main collector plugin"
echo "  web-interface/           # Dashboard web interface"
echo "  start-system.sh          # System startup script"
echo ""
echo "🚀 To get started:"
echo "  1. Edit jira-collector/config.example.json with your Jira credentials"
echo "  2. Run: ./start-system.sh"
echo "  3. Open http://localhost:8080 in your browser"
echo "  4. Run the collector: cd jira-collector && ./jira-collector -config config.example.json"
echo ""
echo "📖 For more information, see the README.md files in each directory"
#!/bin/bash
# -----------------------------------------------------------------------
# Build Script for Aktis Collector Jira
# -----------------------------------------------------------------------

set -euo pipefail

# Configuration
ENVIRONMENT=${1:-dev}
VERSION=""
CLEAN=false
TEST=false
VERBOSE=false
RELEASE=false
DOCKER=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -t|--test)
            TEST=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -r|--release)
            RELEASE=true
            shift
            ;;
        -d|--docker)
            DOCKER=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -e, --environment ENV  Target environment (dev, staging, prod)"
            echo "  -v, --version VER      Version to embed in binary"
            echo "  -c, --clean           Clean build artifacts before building"
            echo "  -t, --test            Run tests before building"
            echo "  --verbose             Enable verbose output"
            echo "  -r, --release         Build optimized release binary"
            echo "  -d, --docker          Build for Docker deployment"
            echo "  -h, --help            Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown parameter: $1"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# Build configuration
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${CYAN}Aktis Collector Jira Build Script${NC}"
echo -e "${CYAN}=================================${NC}"

# Setup paths
PROJECT_ROOT=$(pwd)
VERSION_FILE="$PROJECT_ROOT/.version"
BIN_DIR="$PROJECT_ROOT/bin"
COLLECTOR_OUTPUT="$BIN_DIR/aktis-collector-jira"

# Add .exe extension on Windows
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    COLLECTOR_OUTPUT="$COLLECTOR_OUTPUT.exe"
fi

echo -e "${GRAY}Project Root: $PROJECT_ROOT${NC}"
echo -e "${GRAY}Environment: $ENVIRONMENT${NC}"
echo -e "${GRAY}Git Commit: $GIT_COMMIT${NC}"
echo -e "${GRAY}Build Time: $BUILD_TIME${NC}"

# Handle version management
if [[ -z "$VERSION" ]]; then
    # Try to read version from .version file
    if [[ -f "$VERSION_FILE" ]]; then
        VERSION=$(head -n1 "$VERSION_FILE" | xargs)
        echo -e "${GREEN}Using version from .version file: $VERSION${NC}"
    else
        # Default version
        VERSION="1.0.0-dev"
        echo -e "${YELLOW}Using default version: $VERSION${NC}"
        # Create .version file
        echo "$VERSION" > "$VERSION_FILE"
    fi
else
    echo -e "${GREEN}Using specified version: $VERSION${NC}"
    # Update .version file
    echo "$VERSION" > "$VERSION_FILE"
fi

echo -e "${GREEN}Final Version: $VERSION${NC}"

# Clean build artifacts if requested
if [[ "$CLEAN" == true ]]; then
    echo -e "${YELLOW}Cleaning build artifacts...${NC}"
    if [[ -d "$BIN_DIR" ]]; then
        rm -rf "$BIN_DIR"
    fi
    if [[ -f "go.sum" ]]; then
        rm -f "go.sum"
    fi
fi

# Create bin directory
mkdir -p "$BIN_DIR"

# Run tests if requested
if [[ "$TEST" == true ]]; then
    echo -e "${YELLOW}Running tests...${NC}"
    go test ./... -v
    echo -e "${GREEN}Tests passed!${NC}"
fi

# Download dependencies
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod download

# Build flags
MODULE="aktis-collector-jira/internal/common"
BUILD_FLAGS="-X ${MODULE}.Version=${VERSION} -X ${MODULE}.Build=${BUILD_TIME} -X ${MODULE}.GitCommit=${GIT_COMMIT}"

if [[ "$RELEASE" == true ]]; then
    BUILD_FLAGS="$BUILD_FLAGS -w -s"  # Strip debug info and symbol table
fi

# Build command
echo -e "${YELLOW}Building aktis-collector-jira...${NC}"

export CGO_ENABLED=0
if [[ "$RELEASE" == true ]]; then
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        export GOOS=linux
        export GOARCH=amd64
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        export GOOS=darwin
        export GOARCH=amd64
    elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
        export GOOS=windows
        export GOARCH=amd64
    fi
fi

BUILD_ARGS=("build" "-ldflags=$BUILD_FLAGS" "-o" "$COLLECTOR_OUTPUT" "./cmd/aktis-collector-jira")

if [[ "$VERBOSE" == true ]]; then
    BUILD_ARGS+=("-v")
fi

echo -e "${GRAY}Build command: go ${BUILD_ARGS[*]}${NC}"

go "${BUILD_ARGS[@]}"

# Success message
echo ""
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}Executable: $COLLECTOR_OUTPUT${NC}"

# Show binary info
if [[ -f "$COLLECTOR_OUTPUT" ]]; then
    FILE_SIZE=$(du -h "$COLLECTOR_OUTPUT" | cut -f1)
    echo -e "${GRAY}Size: $FILE_SIZE${NC}"
fi

echo ""
echo -e "${CYAN}Usage examples:${NC}"
echo -e "${GRAY}  $COLLECTOR_OUTPUT -help${NC}"
echo -e "${GRAY}  $COLLECTOR_OUTPUT -version${NC}"
echo -e "${GRAY}  $COLLECTOR_OUTPUT -config config.json${NC}"
echo -e "${GRAY}  $COLLECTOR_OUTPUT -config config.json -update${NC}"
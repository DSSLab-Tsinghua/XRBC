#!/bin/bash

# This script sets up the development environment for CrossRBC project
# ./autoEnv.bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check Go version
check_go_version() {
    if command_exists go; then
        local go_version=$(go version | awk '{print $3}' | sed 's/go//')
        local required_version="1.21.8"
        
        if [[ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" = "$required_version" ]]; then
            print_success "Go version $go_version is compatible (required: $required_version+)"
            return 0
        else
            print_warning "Go version $go_version found, but $required_version+ is recommended"
            return 1
        fi
    else
        print_error "Go is not installed"
        return 1
    fi
}

# Function to install Go
install_go() {
    print_status "Installing Go 1.21.8..."
    
    # Detect OS
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        armv6l) ARCH="armv6l" ;;
        armv7l) ARCH="armv6l" ;;
        *) print_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    GO_VERSION="1.21.8"
    GO_TARBALL="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    GO_URL="https://golang.org/dl/${GO_TARBALL}"
    
    # Download Go
    print_status "Downloading Go from $GO_URL"
    if command_exists wget; then
        wget -q "$GO_URL" -O "/tmp/${GO_TARBALL}"
    elif command_exists curl; then
        curl -L "$GO_URL" -o "/tmp/${GO_TARBALL}"
    else
        print_error "Neither wget nor curl found. Please install one of them."
        exit 1
    fi
    
    # Remove existing Go installation
    if [ -d "/usr/local/go" ]; then
        print_status "Removing existing Go installation"
        sudo rm -rf /usr/local/go
    fi
    
    # Extract Go
    print_status "Extracting Go to /usr/local"
    sudo tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
    
    # Clean up
    rm "/tmp/${GO_TARBALL}"
    
    # Add Go to PATH
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
    fi
    
    print_success "Go installed successfully"
    print_status "Please run 'source ~/.bashrc' or restart your terminal"
}

# Function to verify Go module
verify_go_module() {
    print_status "Verifying Go module..."
    
    if [ ! -f "go.mod" ]; then
        print_error "go.mod file not found. Make sure you're in the correct CrossRBC project directory."
        exit 1
    fi
    
    # Check if this is the correct module
    if grep -q "module CrossRBC" go.mod; then
        print_success "CrossRBC module verified"
    else
        print_error "This doesn't appear to be the CrossRBC module"
        exit 1
    fi
}

# Function to install dependencies
install_dependencies() {
    print_status "Installing Go dependencies..."
    
    # Download dependencies
    go mod download
    
    # Verify dependencies
    go mod verify
    
    # Tidy up dependencies
    go mod tidy
    
    print_success "Dependencies installed and verified"
}

# Function to check if jq is installed
check_jq() {
    if command_exists jq; then
        print_success "jq is already installed: $(jq --version)"
        return 0
    else
        print_warning "jq is not installed"
        return 1
    fi
}

# Function to install jq
# Function to install jq
install_jq() {
    print_status "Installing jq..."
    
    # Detect OS and install accordingly
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        if command_exists apt-get; then
            # Ubuntu/Debian
            print_status "Installing jq using apt-get..."
            sudo apt-get update
            sudo apt-get install -y jq
        elif command_exists yum; then
            # CentOS/RHEL - try EPEL first, then fallback to source
            print_status "Installing jq using yum..."
            
            # First try to install EPEL repository
            if sudo yum install -y epel-release 2>/dev/null; then
                print_status "EPEL repository installed, now installing jq..."
                if sudo yum install -y jq 2>/dev/null; then
                    print_success "jq installed successfully via EPEL"
                else
                    print_warning "EPEL jq installation failed, trying from source..."
                    install_jq_from_source
                fi
            else
                print_warning "EPEL installation failed, trying from source..."
                install_jq_from_source
            fi
        elif command_exists dnf; then
            # Fedora
            print_status "Installing jq using dnf..."
            sudo dnf install -y jq
        elif command_exists pacman; then
            # Arch Linux
            print_status "Installing jq using pacman..."
            sudo pacman -S --noconfirm jq
        else
            # Try to install from source
            install_jq_from_source
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command_exists brew; then
            print_status "Installing jq using Homebrew..."
            brew install jq
        else
            print_error "Homebrew not found. Please install Homebrew first or install jq manually."
            exit 1
        fi
    else
        # Other systems, try to install from source
        install_jq_from_source
    fi
    
    # Verify installation
    if command_exists jq; then
        print_success "jq installed successfully: $(jq --version)"
    else
        print_error "jq installation failed"
        exit 1
    fi
}


# Function to install jq from source
install_jq_from_source() {
    print_status "Installing jq from source..."
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) JQ_ARCH="linux64" ;;
        aarch64) JQ_ARCH="linux64" ;;
        armv6l) JQ_ARCH="linux32" ;;
        armv7l) JQ_ARCH="linux32" ;;
        *) print_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    JQ_VERSION="1.6"
    JQ_URL="https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-${JQ_ARCH}"
    
    # Try multiple download methods and sources
    print_status "Downloading jq from $JQ_URL"
    
    # First try with wget
    if command_exists wget; then
        if wget --timeout=30 --tries=3 -q "$JQ_URL" -O "/tmp/jq"; then
            print_success "Downloaded jq successfully with wget"
        else
            print_warning "wget download failed, trying curl..."
            if command_exists curl; then
                if curl --connect-timeout 30 --retry 3 -L "$JQ_URL" -o "/tmp/jq"; then
                    print_success "Downloaded jq successfully with curl"
                else
                    print_warning "curl download also failed, trying alternative source..."
                    # Try alternative download URL
                    ALT_JQ_URL="https://github.com/jqlang/jq/releases/download/jq-${JQ_VERSION}/jq-${JQ_ARCH}"
                    if curl --connect-timeout 30 --retry 3 -L "$ALT_JQ_URL" -o "/tmp/jq"; then
                        print_success "Downloaded jq from alternative source"
                    else
                        print_error "All download methods failed. Please check your internet connection."
                        print_error "You can manually download jq from: $JQ_URL"
                        print_error "Or install it manually with: sudo yum install epel-release && sudo yum install jq"
                        exit 1
                    fi
                fi
            else
                print_error "Neither wget nor curl found. Please install one of them."
                exit 1
            fi
        fi
    elif command_exists curl; then
        if curl --connect-timeout 30 --retry 3 -L "$JQ_URL" -o "/tmp/jq"; then
            print_success "Downloaded jq successfully with curl"
        else
            print_warning "curl download failed, trying alternative source..."
            # Try alternative download URL
            ALT_JQ_URL="https://github.com/jqlang/jq/releases/download/jq-${JQ_VERSION}/jq-${JQ_ARCH}"
            if curl --connect-timeout 30 --retry 3 -L "$ALT_JQ_URL" -o "/tmp/jq"; then
                print_success "Downloaded jq from alternative source"
            else
                print_error "All download methods failed. Please check your internet connection."
                print_error "You can manually download jq from: $JQ_URL"
                print_error "Or install it manually with: sudo yum install epel-release && sudo yum install jq"
                exit 1
            fi
        fi
    else
        print_error "Neither wget nor curl found. Please install one of them."
        exit 1
    fi
    
    # Verify the downloaded file
    if [ ! -f "/tmp/jq" ] || [ ! -s "/tmp/jq" ]; then
        print_error "Downloaded file is empty or doesn't exist"
        exit 1
    fi
    
    # Make it executable and move to /usr/local/bin
    chmod +x "/tmp/jq"
    
    # Check if /usr/local/bin is writable, if not, try alternative location
    if sudo mv "/tmp/jq" "/usr/local/bin/jq" 2>/dev/null; then
        print_success "jq installed to /usr/local/bin/jq"
    else
        # Try alternative location
        if sudo mv "/tmp/jq" "/usr/bin/jq" 2>/dev/null; then
            print_success "jq installed to /usr/bin/jq"
        else
            print_error "Failed to install jq to system path"
            exit 1
        fi
    fi
    
    print_success "jq installed from source"
}

# Function to verify installation
verify_installation() {
    print_status "Verifying installation..."
    
    # Check Go installation
    if ! command_exists go; then
        print_error "Go installation verification failed"
        return 1
    fi
    
    # Check Go version
    print_status "Go version: $(go version)"
    
    # Check module status
    print_status "Module status:"
    go list -m
    
    # Check dependencies
    print_status "Dependencies:"
    go list -m all
    
    # Check if binary exists
    if [ -f "build/crossrbc" ]; then
        print_success "Binary available: build/crossrbc"
    fi
    
    print_success "Installation verification completed"
}

# Function to show usage information
# to-do: after combination, i need to modiy the configuration
show_usage() {
    print_status "CrossRBC Environment Setup Complete!"
    echo ""
    echo "Available commands:"
    echo "  go run .                    - Run the application directly"
    echo "  go build -o build/crossrbc  - Build the binary"
    echo "  ./build/crossrbc            - Run the built binary"
    echo "  go test ./...               - Run all tests"
    echo "  go mod tidy                 - Clean up dependencies"
    echo ""
    echo "Configuration:"
    echo "  - Edit etc/conf.json for application configuration"
    echo "  - Use ./test.bash <nodes> <consensus> to modify configuration"
    echo ""
    echo "Project structure:"
    echo "  - Source code: Current directory"
    echo "  - Configuration: etc/"
    echo "  - Build output: build/"
    echo "  - Logs: logs/ (if applicable)"
}

# Main execution
main() {
    print_status "Setting up CrossRBC environment..."
    
    # Check if Go is installed and compatible
    if ! check_go_version; then
        read -p "Do you want to install Go 1.21.8? (y/n): " install_go_choice
        if [[ $install_go_choice =~ ^[Yy]$ ]]; then
            install_go
            print_status "Please run 'source ~/.bashrc' and then re-run this script"
            exit 0
        else
            print_warning "Continuing with current Go installation..."
        fi
    fi

# Check if jq is installed and auto-install if not
    if ! check_jq; then
        print_status "jq is required for configuration scripts. Installing automatically..."
        install_jq
    fi
    
    # Verify this is the correct Go module
    verify_go_module
    
    # Install dependencies
    install_dependencies
    
    # Verify installation
    verify_installation
    
    # Show usage information
    show_usage
}

# Run main function
main "$@"
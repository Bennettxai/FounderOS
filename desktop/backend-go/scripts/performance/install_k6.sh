#!/bin/bash

# k6 Installation Script
# Cross-platform installation for k6 load testing tool
# Supports: Windows (Chocolatey/winget), Linux (apt/brew), macOS (brew)

set -e

echo "=================================================="
echo "  k6 Load Testing Tool Installation Script"
echo "=================================================="
echo ""

# Function to detect OS
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
        echo "windows"
    else
        echo "unknown"
    fi
}

# Function to check if k6 is already installed
check_k6_installed() {
    if command -v k6 &> /dev/null; then
        echo "✅ k6 is already installed!"
        k6 version
        echo ""
        read -p "Do you want to reinstall/update k6? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Installation cancelled."
            exit 0
        fi
    fi
}

# Installation functions
install_windows() {
    echo "🪟 Windows installation detected"
    echo ""

    # Check for Chocolatey
    if command -v choco &> /dev/null; then
        echo "📦 Installing k6 via Chocolatey..."
        choco install k6 -y
        return 0
    fi

    # Check for winget
    if command -v winget &> /dev/null; then
        echo "📦 Installing k6 via winget..."
        winget install k6
        return 0
    fi

    # Manual installation
    echo "⚠️  Neither Chocolatey nor winget found!"
    echo ""
    echo "Please install k6 manually using one of these methods:"
    echo ""
    echo "1. Using Chocolatey (recommended):"
    echo "   - Install Chocolatey: https://chocolatey.org/install"
    echo "   - Run: choco install k6"
    echo ""
    echo "2. Using winget:"
    echo "   - Run: winget install k6"
    echo ""
    echo "3. Manual download:"
    echo "   - Download from: https://dl.k6.io/msi/k6-latest-amd64.msi"
    echo "   - Run the installer"
    echo ""
    exit 1
}

install_linux() {
    echo "🐧 Linux installation detected"
    echo ""

    # Check for apt (Debian/Ubuntu)
    if command -v apt &> /dev/null; then
        echo "📦 Installing k6 via apt..."
        sudo gpg -k
        sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
        echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
        sudo apt-get update
        sudo apt-get install k6
        return 0
    fi

    # Check for yum (Red Hat/CentOS)
    if command -v yum &> /dev/null; then
        echo "📦 Installing k6 via yum..."
        sudo yum install https://dl.k6.io/rpm/repo.rpm
        sudo yum install k6
        return 0
    fi

    # Check for Homebrew
    if command -v brew &> /dev/null; then
        echo "📦 Installing k6 via Homebrew..."
        brew install k6
        return 0
    fi

    # Manual installation
    echo "⚠️  No supported package manager found!"
    echo ""
    echo "Please install k6 manually:"
    echo "1. Download from: https://github.com/grafana/k6/releases"
    echo "2. Extract and add to PATH"
    echo ""
    exit 1
}

install_macos() {
    echo "🍎 macOS installation detected"
    echo ""

    # Check for Homebrew
    if command -v brew &> /dev/null; then
        echo "📦 Installing k6 via Homebrew..."
        brew install k6
        return 0
    else
        echo "⚠️  Homebrew not found!"
        echo ""
        echo "Please install Homebrew first:"
        echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        echo ""
        echo "Then run: brew install k6"
        echo ""
        exit 1
    fi
}

# Main installation logic
main() {
    check_k6_installed

    OS=$(detect_os)

    case $OS in
        linux)
            install_linux
            ;;
        macos)
            install_macos
            ;;
        windows)
            install_windows
            ;;
        *)
            echo "❌ Unsupported operating system: $OSTYPE"
            echo "Please install k6 manually from: https://k6.io/docs/getting-started/installation/"
            exit 1
            ;;
    esac

    echo ""
    echo "=================================================="
    echo "  ✅ k6 Installation Complete!"
    echo "=================================================="
    echo ""
    k6 version
    echo ""
    echo "Next steps:"
    echo "1. Run load tests: cd scripts/performance && k6 run load_test_osa.js"
    echo "2. See documentation: cat docs/PERFORMANCE_TESTING.md"
    echo ""
}

# Run main function
main

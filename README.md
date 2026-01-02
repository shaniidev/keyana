# 🗝️ Keyana - JavaScript Secret Hunter

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/shaniidev/keyana?style=for-the-badge)](https://github.com/shaniidev/keyana/stargazers)
[![Release](https://img.shields.io/badge/Version-1.0.0-blue?style=for-the-badge)](https://github.com/shaniidev/keyana/releases)

**A fast JavaScript reconnaissance and secret scanning tool for security researchers and bug bounty hunters.**

[Features](#-features) • [Installation](#-installation) • [Usage](#-usage) • [Performance](#-performance) • [Contributing](#-contributing)

</div>

---

## 🎯 What is Keyana?

Keyana is a **JavaScript analysis tool** built for web security professionals. Unlike traditional scanners that treat JavaScript as plain text, Keyana performs context-aware secret detection with high performance.

### Why Keyana?

- 🚀 **100x Faster** than traditional scanners (scans 633 files in < 5 seconds)
- 🧠 **Aho-Corasick Algorithm** for O(1) pattern matching with 856+ secret patterns
- 🎨 **Built-in Beautification** for minified JavaScript (handles source maps)
- 🔍 **Smart Discovery** integrates Katana, Gau, and Waybackurls
- 🎛️ **Interactive Modes** - Choose between fast or deep scanning
- 📊 **Comprehensive Reports** with organized findings by severity

---

## ✨ Features

### 🔎 Secret Detection Engine
- **856 Built-in Patterns**: AWS, Google Cloud, Stripe, Slack, GitHub, and more
- **Template System**: YAML-based patterns with confidence scoring
- **Aho-Corasick Pre-filtering**: Scans files once regardless of pattern count
- **Smart False Positive Filtering**: Eliminates base64 alphabets, fonts, and generic data
- **Entropy Analysis**: High-entropy string detection for unknown secrets

### 🌐 JavaScript Discovery
- **Multi-source Discovery**: Katana (crawling), Gau (archives), Waybackurls (snapshots)
- **Automatic Deduplication**: Smart URL filtering and JavaScript extraction
- **Progress Tracking**: Real-time progress bars for all stages

### 🎨 Code Beautification
- **Source Map Support**: Automatically downloads and applies source maps
- **Minified JS Handling**: Beautifies packed/obfuscated code
- **Caching**: Reuses beautified files across scans

### 🔗 Integrated Scanners
- **Gitleaks**: Git-focused secret scanning
- **JSLuice**: JavaScript endpoint extraction
- **TruffleHog**: Deep credential scanning
- **LinkFinder**: URL and path discovery

### 📈 Performance Optimizations
- **Parallel Processing**: 8-worker pool for file scanning
- **Line Position Caching**: O(log n) line number lookups
- **Conditional Scanning**: Skips generic checks when high-confidence secrets found
- **Mutex-protected Regexes**: Safe concurrent pattern matching

---

## 🔧 Prerequisites

### Required: js-beautify

Keyana requires **js-beautify** for JavaScript beautification. This is an **essential dependency** and must be installed before using Keyana.

#### Installation:

**For Debian/Ubuntu/Kali Linux:**
```bash
sudo apt update
sudo apt install python3-jsbeautifier
```

**For other Linux distributions:**
```bash
pip install jsbeautifier
# or
pip3 install jsbeautifier
```

**For macOS:**
```bash
brew install jsbeautifier
# or
pip3 install jsbeautifier
```

**For Windows:**
```bash
pip install jsbeautifier
```

**Verify installation:**
```bash
js-beautify --version
```

---

## 📦 Installation

### Option 1: Install via Go (Recommended)
```bash
go install github.com/shaniidev/keyana/cmd/keyana@latest
```

### Option 2: Build from Source
```bash
# Clone the repository
git clone https://github.com/shaniidev/keyana.git
cd keyana

# Build and install
go build -o keyana ./cmd/keyana
sudo mv keyana /usr/local/bin/

# Or for Windows
go build -o keyana.exe ./cmd/keyana
```

### Option 3: Download Pre-built Binaries
Download the latest release from [GitHub Releases](https://github.com/shaniidev/keyana/releases)

---

## 🔧 External Tool Dependencies (Optional but Recommended)

Keyana integrates with several external scanners to provide comprehensive JavaScript analysis. While Keyana works without them, installing these tools unlocks additional features:

### 1. **Gitleaks** - Git Secret Scanning
```bash
# Using Go
go install github.com/gitleaks/gitleaks/v8@latest

# Or using Homebrew (macOS/Linux)
brew install gitleaks

# Or download binary from https://github.com/gitleaks/gitleaks/releases
```

### 2. **TruffleHog** - Deep Credential Scanning
```bash
# Using Go
go install github.com/trufflesecurity/trufflehog/v3@latest

# Or using Homebrew (macOS/Linux)
brew install trufflehog

# Or download binary from https://github.com/trufflesecurity/trufflehog/releases
```

### 3. **JSLuice** - JavaScript Endpoint Extraction
```bash
# Using Go
go install github.com/BishopFox/jsluice/cmd/jsluice@latest

# Or download binary from https://github.com/BishopFox/jsluice/releases
```

### 4. **LinkFinder** - URL and Path Discovery
```bash
# Install Python and pip first, then:
pip install linkfinder

# Or using pipx (recommended)
pipx install linkfinder

# For Kali Linux (usually pre-installed)
sudo apt update
sudo apt install linkfinder
```

### Verify Installations
After installing, verify the tools are available:
```bash
gitleaks version
trufflehog --version
jsluice --version
linkfinder --help
```

> **Note**: Keyana will automatically detect and use any installed scanners. If a scanner is not found, Keyana will skip that specific scanner and continue with its built-in detection engine.

---

## 🚀 Quick Start

### Basic Scan
```bash
keyana -d https://example.com
```

### With Custom Concurrency
```bash
keyana -d https://example.com -c 20
```

### Scan from URL List
```bash
keyana -l urls.txt
```

---

## 📖 Usage

### Command-Line Options
```
  -d string
        Target domain (e.g., https://example.com)
  -l string
        File containing list of domains
  -c int
        Concurrency for discovery (default: 20)
  -t int
        Request timeout in seconds (default: 10)
  -s    
        Silent mode (minimal output)
```

### Interactive Workflow
After discovery and beautification, Keyana presents an interactive menu:

```
[+] Total files for scanning: 633
[1] Scan for secrets
[2] Scan for endpoints
[3] Scan for both
[4] Exit
```

For secret scanning, choose between:
- **FAST Mode**: Uses indexed patterns only (Recommended)
- **DEEP Mode**: Includes generic entropy-based checks

---

## ⚡ Performance

### Benchmark: 633 JavaScript Files

| Scanner | Time | CPU Usage |
|---------|------|-----------|
| **Keyana (Fast Mode)** | **< 5s** | **Normal** |
| Keyana (Deep Mode) | ~2 min | High |
| Traditional Scanners | 5+ min | 100% |

### Scalability
Keyana's Aho-Corasick engine scales to **100,000+ patterns** with constant time complexity:
- 856 patterns → **< 5 seconds**
- 10,000 patterns → **< 5 seconds** (projected)
- 100,000 patterns → **< 5 seconds** (projected)

---

## 📂 Output Structure

```
keyana_output/
└── example.com/
    ├── urls/
    │   ├── katana_urls.txt
    │   ├── gau_urls.txt
    │   └── wayback_urls.txt
    ├── js_files/
    │   ├── downloaded/   # Raw minified JavaScript
    │   └── beautified/   # Cleaned, readable code
    ├── reports/
    │   ├── secrets.txt   # All secret findings
    │   └── endpoints.txt # Discovered endpoints
    └── logs/
        └── secrets_scan.log
```

---

## 🎨 Sample Output

### Secrets Report
```
=================================================================
KEYANA - SECRET SCANNING REPORT
=================================================================
Domain: app.solv.finance
Total Secrets Found: 16
Scan Date: 2026-01-02

--- Template (critical) ---
[!] AWS Access Key ID
    File: /path/to/file.js:1234
    Value: AKIA****************
    Detector: Template (critical)

--- Template (high) ---
[!] Stripe API Key
    File: /path/to/config.js:567
    Value: sk_live_****************
    Detector: Template (high)
```

---

## 🛠️ Configuration

### Custom Pattern Templates
Create YAML files in `templates/` directory:

```yaml
name: Custom Secret Scanner
version: 1.0.0
patterns:
  - id: custom-api-key
    name: Custom API Key
    regex: 'custom_[0-9a-f]{32}'
    confidence: 90
    severity: high
    entropy_check: true
    min_entropy: 4.5
```

---

## 🔧 Advanced Usage

### External Scanner Integration
Keyana automatically detects and uses installed scanners:
- `gitleaks` - Git secret scanning
- `jsluice` - JavaScript analysis
- `trufflehog` - Credential scanning

Install them for enhanced detection:
```bash
go install github.com/trufflesecurity/trufflehog/v3@latest
go install github.com/gitleaks/gitleaks/v8@latest
go install github.com/BishopFox/jsluice/cmd/jsluice@latest
```

---

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup
```bash
git clone https://github.com/shaniidev/keyana.git
cd keyana
go mod download
go build ./cmd/keyana
```

### Adding New Patterns
1. Create a YAML file in `templates/`
2. Follow the pattern schema
3. Test with `go test ./internal/scan`
4. Submit a pull request

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- **coregex** - High-performance regex engine
- **Cloudflare Aho-Corasick** - Multi-pattern string matching
- **ProjectDiscovery** - Katana crawler inspiration
- Community contributors for pattern submissions

---

## 📞 Contact & Support

- **Author**: [@shaniidev](https://github.com/shaniidev)
- **Issues**: [GitHub Issues](https://github.com/shaniidev/keyana/issues)
- **Discussions**: [GitHub Discussions](https://github.com/shaniidev/keyana/discussions)

---

<div align="center">

**⭐ If Keyana helped you find secrets, consider giving it a star!**

</div>

# Keyana

A JavaScript secret scanner and reconnaissance tool for security researchers.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/badge/Version-1.0.2-blue?style=flat-square)](https://github.com/shaniidev/keyana/releases)

## What is Keyana?

Keyana scans JavaScript files for secrets, API keys, and credentials. It uses the Aho-Corasick algorithm to match 856+ patterns efficiently and includes built-in JavaScript beautification for minified code.

### Features

- Scans 633 files in under 5 seconds
- 856 built-in secret detection patterns (AWS, GitHub, Stripe, etc.)
- JavaScript discovery via Katana, Gau, and Waybackurls
- Automatic beautification of minified JavaScript
- Parallel processing with 8-worker pool
- Interactive scan modes (Fast/Deep)
- Optional integration with Gitleaks, TruffleHog, JSLuice, and LinkFinder

## Prerequisites

### Required Dependencies

**js-beautify** (required)
```bash
# Debian/Ubuntu/Kali
sudo apt install python3-jsbeautifier

# Other Linux/macOS
pip3 install jsbeautifier

# Windows
pip install jsbeautifier

# Verify
js-beautify --version
```

### Discovery Tools (required for URL discovery)

**Katana**
```bash
go install github.com/projectdiscovery/katana/cmd/katana@latest
```

**Gau (Get All URLs)**
```bash
go install github.com/lc/gau/v2/cmd/gau@latest
```

**Waybackurls**
```bash
go install github.com/tomnomnom/waybackurls@latest
```

### Optional Scanners

**Gitleaks**
```bash
go install github.com/gitleaks/gitleaks/v8@latest
```

**TruffleHog**
```bash
go install github.com/trufflesecurity/trufflehog/v3@latest
```

**JSLuice**
```bash
go install github.com/BishopFox/jsluice/cmd/jsluice@latest
```

**LinkFinder**
```bash
pip install linkfinder
```

## Installation

### Using Go
```bash
go install github.com/shaniidev/keyana/cmd/keyana@latest
```

### Build from Source
```bash
git clone https://github.com/shaniidev/keyana.git
cd keyana
go build -o keyana ./cmd/keyana
```

### Download Binary
Download from [releases](https://github.com/shaniidev/keyana/releases)

## Usage

### Basic Scan
```bash
keyana -d https://example.com
```

### Options
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

### Interactive Mode

After discovery, Keyana presents scan options:
```
[+] Total files for scanning: 633
[1] Scan for secrets
[2] Scan for endpoints
[3] Scan for both
[4] Exit
```

For secret scanning:
- **FAST Mode**: Uses indexed patterns only (recommended)
- **DEEP Mode**: Includes entropy-based detection

## Performance

| Scanner | Time | CPU Usage |
|---------|------|-----------|
| Keyana (Fast) | < 5s | Normal |
| Keyana (Deep) | ~2 min | High |

## Output Structure

```
keyana_output/
└── example.com/
    ├── urls/
    │   ├── katana_urls.txt
    │   ├── gau_urls.txt
    │   └── wayback_urls.txt
    ├── js_files/
    │   ├── downloaded/
    │   └── beautified/
    ├── reports/
    │   ├── secrets.txt
    │   └── endpoints.txt
    └── logs/
        └── secrets_scan.log
```

## Configuration

### Custom Patterns

Create YAML files in `templates/` directory:

```yaml
name: Custom Scanner
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

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE)

## Contact

- GitHub: [@shaniidev](https://github.com/shaniidev)
- Issues: [GitHub Issues](https://github.com/shaniidev/keyana/issues)

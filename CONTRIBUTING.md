# Contributing to Keyana

First off, thank you for considering contributing to Keyana! 🎉

## 🤝 How Can I Contribute?

### Reporting Bugs
- Use the GitHub issue tracker
- Describe the bug in detail
- Include steps to reproduce
- Mention your OS and Go version

### Suggesting Features
- Open an issue with the `enhancement` label
- Explain the use case
- Describe the expected behavior

### Submitting Pattern Templates
We're always looking for new secret patterns!

1. Create a YAML file in `templates/`
2. Follow this schema:
```yaml
name: Your Scanner Name
version: 1.0.0
author: Your Name
category: api-keys
patterns:
  - id: unique-pattern-id
    name: Human Readable Name
    regex: 'your_regex_here'
    confidence: 90
    severity: high
    entropy_check: true
    min_entropy: 4.5
    tags:
      - api
      - credentials
```
3. Test thoroughly with false positive checks
4. Submit a PR

### Code Contributions

#### Development Setup
```bash
git clone https://github.com/shaniidev/keyana.git
cd keyana
go mod download
go build ./cmd/keyana
```

#### Code Style
- Follow standard Go conventions
- Run `gofmt` before committing
- Add comments for exported functions
- Write tests for new features

#### Pull Request Process
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 Commit Message Guidelines

Format:
```
<emoji> <type>: <subject>

<body>
```

Types:
- `✨ feat`: New feature
- `🐛 fix`: Bug fix
- `📚 docs`: Documentation
- `🎨 style`: Code style/formatting
- `♻️ refactor`: Code refactoring
- `⚡ perf`: Performance improvement
- `✅ test`: Adding tests

Example:
```
✨ feat: Add GitHub token pattern detection

Added new pattern template for detecting GitHub personal access tokens
with entropy validation and false positive filtering.
```

## 🧪 Testing

Run tests before submitting:
```bash
go test ./...
```

Add tests for new features:
```bash
go test ./internal/scan -v
```

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.

## 💬 Questions?

Feel free to open a discussion on GitHub or reach out to @shaniidev

Thank you for contributing! 🚀

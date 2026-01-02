# Contributing to Keyana

Thank you for your interest in contributing to Keyana.

## Bug Reports

Use the GitHub issue tracker to report bugs. Include:
- Description of the bug
- Steps to reproduce
- Your OS and Go version
- Expected vs actual behavior

## Feature Requests

Open an issue with the `enhancement` label:
- Describe the feature
- Explain the use case
- Example of expected behavior

## Contributing Patterns

We welcome new secret detection patterns.

### Pattern Submission

1. Create a YAML file in `templates/`
2. Use this format:

```yaml
name: Scanner Name
version: 1.0.0
author: Your Name
category: api-keys
patterns:
  - id: unique-id
    name: Pattern Name
    regex: 'your_regex_here'
    confidence: 90
    severity: high
    entropy_check: true
    min_entropy: 4.5
    tags:
      - api
      - credentials
```

3. Test for false positives
4. Submit a pull request

## Code Contributions

### Setup

```bash
git clone https://github.com/shaniidev/keyana.git
cd keyana
go mod download
go build ./cmd/keyana
```

### Style Guidelines

- Follow Go conventions
- Run `gofmt` before committing
- Add comments for exported functions
- Write tests for new features

### Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Commit changes: `git commit -m 'Add your feature'`
4. Push to branch: `git push origin feature/your-feature`
5. Open a pull request

## Commit Messages

Format:
```
<type>: <subject>

<body>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code formatting
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `test`: Adding tests

Example:
```
feat: Add GitHub token pattern detection

Added pattern template for GitHub personal access tokens
with entropy validation and false positive filtering.
```

## Testing

Run tests before submitting:
```bash
go test ./...
```

Test specific packages:
```bash
go test ./internal/scan -v
```

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions

Open a discussion on GitHub or contact @shaniidev

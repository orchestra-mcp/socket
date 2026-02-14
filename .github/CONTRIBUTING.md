# Contributing to Orchestra WebSocket Plugin

Contributions are **welcome** and will be fully **credited**.

## Development Setup

```bash
# From the plugin directory
cd plugins/socket

# Run tests
go test ./tests/... -v

# Lint
golangci-lint run ./...

# Format
gofumpt -w config/ providers/ src/ tests/
```

Or from the monorepo root:

```bash
make check    # format + lint + tests for everything
```

### Requirements

- Go 1.24+
- golangci-lint
- gofumpt

## Pull Request Process

1. **Fork** and branch from `main`.
2. **Write tests** for new features or behavior changes.
3. **Run `golangci-lint run ./...`** — zero errors required.
4. **One PR per feature.**
5. **Update docs** if changing behavior.
6. **Follow [SemVer v2.0.0](https://semver.org/)** — do not break public APIs.

**Happy coding!**

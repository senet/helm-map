# Contributing to helm-map

Thank you for your interest in contributing to helm-map!

## Development Setup

1. Install Go 1.22+
2. Clone the repo:
   ```bash
   git clone https://github.com/senet/helm-map.git
   cd helm-map
   ```
3. Build:
   ```bash
   make build
   ```
4. Run tests:
   ```bash
   make test
   ```

## Code Style

- Run `make lint` before submitting PRs
- Follow standard Go conventions
- Add tests for new functionality

## Pull Requests

1. Fork the repo
2. Create a feature branch from `main`
3. Add tests for your changes
4. Ensure `make test` and `make lint` pass
5. Submit a PR with a clear description

## Reporting Issues

Open an issue on GitHub with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behaviour
- helm-map version (`helm map version`)

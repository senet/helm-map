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

All PRs require:
- Passing CI (lint + test + build)
- Approval from the repository owner ([@senet](https://github.com/senet))
- Review from a [CODEOWNER](/.github/CODEOWNERS)

## Branch Protection

The `main` branch is protected:
- Direct pushes are not allowed
- All changes must go through a pull request
- CI status checks must pass before merging
- Force pushes and branch deletion are blocked

## Reporting Issues

Open an issue on GitHub with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behaviour
- helm-map version (`helm map version`)

## Security

To report a security vulnerability, see [SECURITY.md](/.github/SECURITY.md).
**Do not open a public issue for security vulnerabilities.**

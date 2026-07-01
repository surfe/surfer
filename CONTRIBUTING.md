# Contributing to surfer

Thanks for your interest in improving `surfer`! This document covers how to get set
up and the conventions we follow.

## Prerequisites

- Go (see the version in [`go.mod`](go.mod))
- `make`

## Getting started

```bash
git clone https://github.com/Surfe/surfer.git
cd surfer
make build        # builds ./surfer
./surfer --help
```

## Development workflow

```bash
go build ./...    # compile
go test ./...     # run tests
make lint         # golangci-lint (config in .golangci.yml)
make install      # install to ~/.local/bin/surfer
```

Always run `go build ./... && go test ./...` before opening a PR.

## Conventions

- The command tree is **verb-first**: `surfer search|enrich companies|people`.
  New commands should follow the existing `cmd/*.go` patterns.
- Surfe API request/response shapes are strict — see [AGENTS.md](AGENTS.md) for the
  exact v2 request bodies (a wrong shape returns HTTP 500).
- Add or update tests for any behavior change.
- Keep commits focused. Commit messages follow
  [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`,
  `refac:`, `docs:`, `test:`, `ci:`) — these drive the generated changelog.

## Pull requests

1. Fork and create a feature branch.
2. Make your change with tests; ensure `go build ./...`, `go test ./...`, and
   `make lint` all pass.
3. Open a PR against `main` with a clear description.

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE).

## Reporting bugs & security issues

- Bugs / features: open a GitHub issue.
- Security vulnerabilities: see [SECURITY.md](SECURITY.md) (do **not** open a public issue).

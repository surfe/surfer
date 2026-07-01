# Changelog

All notable changes to this project are documented here. Release notes are generated
by [GoReleaser](https://goreleaser.com/) from Conventional Commit messages; this file
tracks notable unreleased changes.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Verb-first command tree: `surfer search companies|people` and
  `surfer enrich companies|people`, with enrichment status under
  `surfer enrich <entity> status <id>`.
- `surfer whoami` — show the authenticated account (decoded locally from the token).
- `--wait` flag for enrichment commands to poll until completion (with `--poll-interval`
  and `--timeout`).
- Open-source project files: `LICENSE` (MIT), `CONTRIBUTING.md`, `SECURITY.md`,
  `CODE_OF_CONDUCT.md`, CI and release workflows, Dependabot.
- Homebrew distribution via the `surfe/tap` tap (`brew install surfe/tap/surfer`),
  published automatically by GoReleaser on each tagged release.

### Fixed
- `companies search` now wraps filters in the required `filters` object and uses
  `employeeCount`/`from`/`to`, fixing an API 500.
- `enrich companies --json` no longer requires `--domain` (removed an unreachable
  required-flag check).

### Changed
- API calls now target `api.surfe.com` (configurable via `api-url` / `SURFER_API_URL`);
  OAuth uses `auth-url` / `SURFER_AUTH_URL`.
- OAuth login now sends and validates a `state` parameter; auth HTTP calls have timeouts.

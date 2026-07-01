# AGENTS.md

Guidance for AI coding agents (Claude Code, Codex, etc.) working in this repo.
Human contributors: see [CONTRIBUTING.md](CONTRIBUTING.md).

## What this is

`surfer` is a command-line client for the [Surfe](https://surfe.com) public API
(search & enrich people and companies). Go + [Cobra](https://github.com/spf13/cobra).
Module path: `github.com/Surfe/surfer`.

## Build / test / lint

```bash
make build        # build ./surfer (with version ldflags)
make install      # build + copy to ~/.local/bin/surfer
go build ./...    # compile everything
go test ./...     # run all tests
make lint         # golangci-lint
```

Always verify with the real compiler before claiming done: `go build ./... && go test ./...`.
Re-run `make install` to push changes onto your PATH (the repo `./surfer` and the
installed binary are separate copies).

## Layout

- `cmd/` — Cobra commands (one file per command). Command tree is **verb-first**.
- `internal/auth/` — OAuth 2.0 + PKCE login, token storage (`~/.surfer/tokens.json`, mode 0600), refresh.
- `internal/client/` — thin HTTP client; injects the bearer token, marshals JSON.
- `pkg/config/` — viper config (file `~/.surfer/config.yaml`, env `SURFER_*`).
- `pkg/output/` — `output.Print` renders json (default) or csv.
- `pkg/version/` — self-update version check.

## Command structure (verb-first)

```
surfer search companies|people [flags]
surfer enrich companies|people [flags]            # async; --wait to block until done
surfer enrich companies|people status <id>        # poll/retrieve an enrichment
surfer whoami                                     # show authenticated account (local JWT decode)
surfer credits | login | logout | version | update | ai
```

## Conventions & gotchas (learned the hard way)

- **Surfe v2 request shapes matter — a wrong shape returns HTTP 500, not 400.**
  - `POST /v2/companies/search`: filters MUST be nested under a required top-level
    `filters` object; employee filter is `employeeCount` with `from`/`to` (NOT
    `employeesCount`/`min`/`max`).
  - `POST /v2/people/search`: NO `filters` wrapper — top-level `people` and `companies` objects.
  - `POST /v2/companies/enrich`: `{"companies":[{"domain","externalID"}]}` (field is `companies`, not `organizations`).
  - `POST /v2/people/enrich`: `{"people":[...],"include":{email,mobile,linkedInUrl,jobHistory}}` (`include` required, ≥1 field).
  - Docs: https://developers.surfe.com
- **Do not use `MarkFlagRequired` on a command that also accepts `--json`** — Cobra
  validates required flags before `RunE`, which makes the `--json` branch unreachable.
  Enforce required flags manually in `RunE` on the non-JSON path.
- **Output**: feed `output.Print` a `map[string]any` (round-trip structs through JSON)
  so json and csv both work. Keep records flat — the CSV writer treats the first array
  field as the row set, so a nested array on a single-object record breaks CSV.
- **Endpoints**: API → `api.surfe.com` (`api-url` / `SURFER_API_URL`); auth/OAuth →
  `eu.prod.surfe.com` (`auth-url` / `SURFER_AUTH_URL`). Auth also accepts `SURFE_API_KEY`.
- Enrichment is async: the start call returns an `enrichmentID`; poll the `status`
  subcommand, or pass `--wait`.

## Style

Match the surrounding code. Keep commands small and consistent with the existing
`cmd/*.go` files. Add/maintain tests for new behavior.

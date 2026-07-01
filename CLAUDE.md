# CLAUDE.md

This project uses [`AGENTS.md`](AGENTS.md) as the single source of truth for AI
coding agents. Please read it for build/test commands, repo layout, the verb-first
command structure, and the Surfe API conventions and gotchas.

Quick reminders:
- Verify with the real compiler before claiming done: `go build ./... && go test ./...`
- Command tree is verb-first: `surfer search|enrich companies|people`.
- Wrong Surfe v2 request shapes return HTTP 500 — see AGENTS.md for the exact shapes.

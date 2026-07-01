```
   _____ __  ______  ________________
  / ___// / / / __ \/ ____/ ____/ __ \
  \__ \/ / / / /_/ / /_  / __/ / /_/ /
 ___/ / /_/ / _, _/ __/ / /___/ _, _/
/____/\____/_/ |_/_/   /_____/_/ |_|
```

# Surfer CLI

The official command-line interface for the [Surfe API](https://surfe.com). Search contacts, enrich leads with verified emails and phone numbers, and manage your Surfe account — all from the terminal.

Built for developers, sales engineers, and AI agents that need programmatic access to Surfe's data enrichment platform.

## Installation

### Homebrew (recommended)

```bash
brew install surfe/tap/surfer-cli
```

The formula is `surfer-cli` (to avoid a name clash with an unrelated
`surfer` in homebrew-core), but it installs the **`surfer`** command:

```bash
surfer version
```

To upgrade later: `brew upgrade surfer-cli`. (First time only, you can also run
`brew tap surfe/tap` and then `brew install surfer-cli`.)

### From GitHub Releases

Download the latest binary for your platform from the [Releases](https://github.com/Surfe/surfer/releases) page.

**macOS (Apple Silicon):**
```bash
curl -sL https://github.com/Surfe/surfer/releases/latest/download/surfer_darwin_arm64.tar.gz | tar xz
sudo mv surfer /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -sL https://github.com/Surfe/surfer/releases/latest/download/surfer_darwin_amd64.tar.gz | tar xz
sudo mv surfer /usr/local/bin/
```

**Linux (amd64):**
```bash
curl -sL https://github.com/Surfe/surfer/releases/latest/download/surfer_linux_amd64.tar.gz | tar xz
sudo mv surfer /usr/local/bin/
```

### From source

```bash
git clone https://github.com/Surfe/surfer.git
cd surfer
make install
```

Requires Go 1.26+.

## Quick start

### 1. Authenticate

```bash
surfer login
```

This opens your browser to sign in via Surfe's OAuth 2.0 flow. Tokens are stored locally in `~/.surfer/tokens.json`.

### 2. Search for people

```bash
surfer search people --company-domains "stripe.com" --job-titles "CTO,VP Engineering"
```

### 3. Enrich a contact

```bash
surfer enrich people --linkedin "https://linkedin.com/in/johndoe"
```

### 4. Check your credits

```bash
surfer credits
```

## Commands

### Authentication

| Command | Description |
|---------|-------------|
| `surfer login` | Sign in via browser (OAuth 2.0 + PKCE) |
| `surfer logout` | Sign out and clear local tokens |
| `surfer whoami` | Show the account you're authenticated as |

### People

| Command | Description |
|---------|-------------|
| `surfer search people` | Search contacts by job title, company, location |
| `surfer enrich people` | Get verified emails, phones, LinkedIn profiles (add `--wait` to block until done) |
| `surfer enrich people status <id>` | Check enrichment progress |

### Companies

| Command | Description |
|---------|-------------|
| `surfer search companies [names...]` | Search companies by name, industry, size |
| `surfer enrich companies` | Get firmographic data for a company (add `--wait` to block until done) |
| `surfer enrich companies status <id>` | Check enrichment progress |

### Account

| Command | Description |
|---------|-------------|
| `surfer credits` | Check remaining email and mobile credits |
| `surfer version` | Print CLI version |
| `surfer update` | Self-update to the latest release |

## Usage examples

### Search people at a company

```bash
surfer search people --company-domains "google.com" --seniorities "C-Level,VP" --limit 5
```

### Search by job title across multiple countries

```bash
surfer search people --job-titles "Head of Sales" --countries "US,UK,DE" --limit 20
```

### Enrich a person by LinkedIn URL

```bash
surfer enrich people --linkedin "https://linkedin.com/in/janedoe" --include-mobile
```

### Enrich a person by name and company

```bash
surfer enrich people --first-name Jane --last-name Doe --company-domain google.com
```

### Search companies by domain

```bash
surfer search companies --domains "stripe.com,shopify.com"
```

### Search companies by name

```bash
surfer search companies apple google microsoft
```

### Pass a raw JSON payload

Every search and enrich command supports `--json` for full control over the request body:

```bash
surfer search people --json '{
  "companies": {"domains": ["surfe.com"]},
  "people": {"jobTitles": ["CTO"]},
  "limit": 5
}'
```

```bash
surfer enrich people --json '{
  "people": [{"linkedinUrl": "https://linkedin.com/in/johndoe"}],
  "include": {"email": true, "mobile": true}
}'
```

### Pipe output to other tools

Surfer outputs JSON, so you can pipe to `jq`, `gron`, or any JSON processor:

```bash
# Get all emails from search results
surfer search people --company-domains "surfe.com" | jq '.people[].linkedInUrl'

# Pretty-print company data
surfer search companies --domains "stripe.com" | jq .
```

## AI agents & automation

Surfer is designed to be used by AI agents, LLM tool-calling pipelines, and automation scripts. Every command:

- Returns structured **JSON** to stdout
- Accepts `--json` for raw API payloads (no flag parsing needed)
- Uses exit code `0` for success, `1` for errors
- Supports **non-interactive auth** via `SURFE_API_KEY` environment variable

### Server-to-server authentication

For CI/CD, scripts, and AI agents that can't open a browser:

```bash
export SURFE_API_KEY="your-api-key"
surfer search people --company-domains "example.com"
```

When `SURFE_API_KEY` is set, the CLI skips the OAuth flow and uses the API key directly.

### Example: AI agent tool definition

```json
{
  "name": "surfe_search_people",
  "description": "Search for business contacts by company, job title, or location",
  "parameters": {
    "json": {
      "type": "string",
      "description": "Raw JSON payload for POST /v2/people/search"
    }
  },
  "command": "surfer search people --json '{json}'"
}
```

## Configuration

Surfer looks for configuration in this order (highest priority first):

1. **Environment variables** (`SURFER_AUTH_URL`, `SURFER_CLIENT_ID`)
2. **Config file** (`~/.surfer/config.yaml`)
3. **Built-in defaults**

### Config file

```yaml
# ~/.surfer/config.yaml
auth-url: https://eu.prod.surfe.com
client-id: hubspot
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SURFE_API_KEY` | — | API key for non-interactive auth (skips OAuth) |
| `SURFER_AUTH_URL` | `https://eu.prod.surfe.com` | API and auth base URL |
| `SURFER_CLIENT_ID` | `hubspot` | OAuth client ID |

### Debug mode

Add `--debug` to any command to see the full HTTP request and response:

```bash
surfer --debug search people --company-domains "surfe.com"
```

## Data directory

All local data is stored in `~/.surfer/` (or `$XDG_CONFIG_HOME/surfer/`):

```
~/.surfer/
  config.yaml          # Configuration (optional)
  tokens.json          # OAuth tokens (auto-managed)
  version-check.json   # Update check cache
```

## Updating

### Self-update

```bash
surfer update
```

This checks GitHub Releases for a newer version, downloads it, verifies the checksum, and replaces the binary in-place.

### Automatic update notifications

After every command, Surfer checks for new versions in the background (cached for 24 hours). If a newer version is available, you'll see:

```
🚀 A new version is available: v1.2.0 → Run 'surfer update' to upgrade.
```

## Building from source

```bash
make build          # Compile binary
make install        # Build + install to ~/.local/bin
make test           # Run tests with coverage
make lint           # Run linter
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) and our
[Code of Conduct](CODE_OF_CONDUCT.md) before opening a pull request. To report a
security issue, see [SECURITY.md](SECURITY.md).

## License

Licensed under the [MIT License](LICENSE). Copyright © 2026 Surfe.

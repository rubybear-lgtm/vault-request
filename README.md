# vault-request

One-time secret request links for OpenClaw agents. A lightweight Go + htmx tool that generates a temporary local HTTP server with a single-use form for collecting sensitive values.

**The agent never sees the secret.** The agent generates a link, you fill in the value, and it gets written straight to a `.env` file. The agent only knows whether it succeeded or not.

## How it works

```
Agent runs:      vault request GEMINI_API_KEY --note "Google AI Studio key"
                 → http://127.0.0.1:34291/claim/abc123xyz

User opens link  → htmx-powered form with one password field
User submits     → value written to .env, server shuts down
Agent knows      → "Secret provisioned ✅" (binary signal only)
```

The link is:
- **One-time** — single use, invalidated after first submission
- **Local only** — binds to `127.0.0.1` by default
- **Time-limited** — expires after TTL (default 30 min)
- **Token-authenticated** — 32-byte random hex token in the URL path

## Usage

```bash
# Basic
vault request GEMINI_API_KEY

# With note and custom output
vault request GEMINI_API_KEY --note "Google AI Studio key" --out config/secrets.env

# JSON output for agent parsing
vault request GEMINI_API_KEY --json

# Specific port (default: random)
vault request GEMINI_API_KEY --port 9999

# Short TTL
vault request GEMINI_API_KEY --ttl 5
```

## Output

The secret is saved to a `.env` file (default: `.env`):

```env
GEMINI_API_KEY="AIzaSy..."
```

The value is shell-escaped. Existing keys are overwritten in place.

## Build

```bash
go build -o vault .
```

Requires Go 1.24+.

## Project structure

```
├── main.go              # Entry point
├── cmd/request.go       # CLI subcommand
├── server/
│   ├── server.go        # HTTP server lifecycle
│   └── handlers.go      # htmx form + handlers
├── token/token.go       # One-time token generation
├── store/store.go       # Store interface + .env writer
└── vault_test.go        # Integration tests
```

## Tests

```bash
go test -v -count=1 ./...
```

## License

MIT

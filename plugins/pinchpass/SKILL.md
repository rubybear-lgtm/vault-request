---
name: pinchpass
description: Generate one-time E2E-encrypted secret request links via pinchpass. Use when an agent needs to collect API keys, tokens, passwords, connection strings, or any sensitive value from the user. Trigger on keywords: API key, secret, token, credentials, password, .env.
version: 1.0.0
homepage: https://github.com/rubybear-lgtm/PinchPass
metadata: { "openclaw": { "requires": { "bins": ["pinchpass"] } } }
---

# pinchpass

Securely collect sensitive values from users with end-to-end encryption. The
relay never sees the secret — encryption happens in the browser, the key lives
only in the URL fragment.

## When to use

- An agent needs a value it does not have (API key, token, password, etc.)
- The user says "I need to give you a secret" or "where do I put my key?"
- A `.env` file needs to be populated with credentials

Do **not** use for non-sensitive plaintext — there is no reason to go through
E2E encryption for a public value.

## CLI

```bash
pinchpass request <secret-name>... [flags]
```

Bare `pinchpass <name>` also works (auto-detects missing subcommand).

### Flags

| Flag          | Default       | Description                              |
|---------------|---------------|------------------------------------------|
| `-tunnel`     | false         | Open a bore.pub tunnel for a public URL  |
| `-note`       | —             | Description shown on the form            |
| `-out`        | `.env`        | Output .env file path                    |
| `-ttl`        | 30            | Minutes until the link expires           |
| `-port`       | random        | Local port to bind                       |
| `-listen-addr`| `127.0.0.1`   | Address to listen on                     |
| `-json`       | false         | Machine-readable JSON output             |

## Workflow

1. Run `pinchpass request <name> -json` to get a link
2. Give the link to the user (the fragment `#k=<hex>` is the decryption key)
3. User opens the link in a browser, fills in the value, submits
4. Server waits for the submission or TTL expiry
5. On success, the value is decrypted and written to `.env`

### JSON output mode

Use `-json` for machine parsing. The JSON is printed twice:
- **Before waiting**: `{ "success": true, "url": "...", "port": N }`
- **After completion**: `{ "success": true/false, "names": [...], "message": "..." }`

## Workflow example

```bash
# Collect an API key via public tunnel
pinchpass request GEMINI_API_KEY -tunnel -json
# → Prints link object with url field
# → Agent gives url to user
# → Agent waits for result
# → On completion, value in .env
```

## Edge cases

- **One-time use**: after the first POST, the token is consumed. Retry
  generates a 404. If it fails, generate a new link.
- **TTL expiry**: server shuts down after the TTL. No value is saved and exit
  code is 1. Generate a new link with a longer `-ttl` if needed.
- **No tunnel**: without `-tunnel`, the link is only reachable on
  `127.0.0.1:<port>`. Use for local-only workflows or when the user is on the
  same machine.
- **Bore.pub unreachable**: tunnel startup fails if bore.pub is down. Fall
  back to local mode or retry.
- **Shell escaping**: values written to `.env` are shell-escaped for `\`,
  `"`, `$`, and backtick.

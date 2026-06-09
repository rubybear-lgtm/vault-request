# vault-request: Bore Tunnel + E2E Encryption + UI Redesign

**Date:** 2026-06-09
**Status:** Approved

## Overview

Three coordinated changes to vault-request:

1. **Bore tunnel** (`--tunnel` flag) — native Go bore.pub client so agents can share secret request links outside the local network.
2. **Always-on E2E encryption** — AES-256-GCM with key in URL fragment; server and relay never see plaintext.
3. **UI redesign** — Solarized Light palette, ASCII art header, monospace throughout.

## Architecture

```
Agent CLI                 Browser                  bore.pub relay
─────────────────────     ──────────────────────   ──────────────
1. gen 32-byte key (K)
2. start HTTP server
3. (--tunnel) bore dial → bore.pub assigns port N
4. print URL:
   http://bore.pub:N/claim/<token>#k=<K>

                          5. user opens URL
                             reads K from fragment
                             (fragment never sent over wire)
                             encrypts secret with K (WebCrypto)
                             POSTs base64(iv ‖ ciphertext)    ──→ relay proxies
                                                                   encrypted bytes
6. Wait() returns
   (true, encryptedBlob)
7. agent decrypts with K
8. store.Save() → .env
9. bore stops
```

## Structural changes

| File | Change |
|---|---|
| `tunnel/bore.go` | New package — bore.pub client |
| `server/server.go` | Drop `Store` from `Config`; add `encryptedBlob []byte`; `Wait()` → `(bool, []byte)` |
| `server/handlers.go` | Inline WebCrypto JS in form; Solarized Light + ASCII redesign |
| `cmd/request.go` | Key gen; `-tunnel` flag; decrypt blob after `Wait()`; call `store.Save()` |
| `token/token.go` | No change — `Generate()` reused for encryption key |
| `store/store.go` | No change |
| `vault_test.go` | Update `Wait()` call sites; encryption round-trip test |

**New dependencies: zero.** Everything uses Go stdlib.

## E2E Encryption

### Key generation

```go
// cmd/request.go — reuse token.Generate() which is crypto/rand 32 bytes → hex
encKeyHex, err := token.Generate()  // 64-char hex = 32 bytes
keyBytes, _    := hex.DecodeString(encKeyHex)
```

### URL construction

```
http://<host>:<port>/claim/<token>#k=<encKeyHex>
```

The `#k=<key>` fragment is stripped by browsers before sending HTTP requests. bore.pub relay never sees it.

### Browser (inline JS, no CDN)

```javascript
async function getKey() {
    const hex = location.hash.replace(/^#k=/, '');
    if (!hex || hex.length !== 64) return null;
    const bytes = new Uint8Array(hex.match(/.{2}/g).map(b => parseInt(b, 16)));
    return crypto.subtle.importKey('raw', bytes, {name: 'AES-GCM'}, false, ['encrypt']);
}

async function encryptValue(key, plaintext) {
    const iv  = crypto.getRandomValues(new Uint8Array(12));
    const ct  = await crypto.subtle.encrypt({name: 'AES-GCM', iv}, key,
                    new TextEncoder().encode(plaintext));
    const buf = new Uint8Array(12 + ct.byteLength);
    buf.set(iv);
    buf.set(new Uint8Array(ct), 12);
    return btoa(String.fromCharCode(...buf));
}
```

If the fragment is absent or malformed the form is disabled with an inline error — no silent failure.

### Server

- Receives `base64(iv ‖ ciphertext)` as form field `value`
- Base64-decodes, stores as `s.encryptedBlob []byte`
- Never calls `store.Save()` — that responsibility moves to the CLI
- `Wait()` signature: `func (s *Server) Wait() (bool, []byte)`

### Agent (CLI)

```go
ok, blob := srv.Wait()
if !ok || len(blob) < 13 {
    // timeout or empty — exit 1
}
block, _ := aes.NewCipher(keyBytes)
gcm, _   := cipher.NewGCM(block)
iv, ct   := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
plain, err := gcm.Open(nil, iv, ct, nil)
if err != nil {
    // decryption failed — tampered payload — exit 1
}
store.Save(cfg.SecretName, string(plain))
```

### Error paths

| Condition | Behaviour |
|---|---|
| Fragment missing / wrong length | Form disabled, "link opened without encryption key" message |
| WebCrypto unavailable | Form disabled, "requires a modern browser" message |
| Malformed base64 from client | Server returns 400, JS shows error fragment |
| Decryption failure (tampered) | CLI exits with error, nothing written to `.env` |
| TTL expires | `Wait()` returns `(false, nil)`, CLI exits 1 |

## Bore Tunnel

### Protocol

bore.pub uses **null-byte–delimited JSON** over TCP port 7835. Max frame 256 bytes. No auth on the public relay.

**Client → Server:**
```json
{"Hello":0}            // request random port (0 = let server choose)
{"Accept":"<uuid>"}    // accept a proxied connection on a new TCP conn
```

**Server → Client:**
```json
{"Hello":12345}        // assigned public port
{"Connection":"<uuid>"} // new inbound connection ready
"Heartbeat"            // keepalive — ignore
{"Error":"..."}        // fatal error
```

### Go implementation (`tunnel/bore.go`)

```go
package tunnel

type Tunnel struct {
    localPort  int
    remotePort int
    conn       net.Conn
    br         *bufio.Reader
    stopOnce   sync.Once
    stopCh     chan struct{}
}

func Start(localPort int) (*Tunnel, error)
    // 1. TCP connect bore.pub:7835
    // 2. send {"Hello":0}
    // 3. recv {"Hello":N} → t.remotePort = N
    // 4. start event loop goroutine
    // return t

func (t *Tunnel) RemotePort() int
func (t *Tunnel) Stop()

// event loop (goroutine):
//   recv message loop
//   Connection(id) → go t.proxy(id)
//   Heartbeat      → continue
//   Error / EOF    → return

// proxy(id) goroutine:
//   1. TCP connect bore.pub:7835 (new conn)
//   2. send {"Accept":"<id>"}
//   3. TCP connect 127.0.0.1:<localPort>
//   4. io.MultiReader(brBuf, remoteConn) to drain framing buffer
//   5. io.Copy bidirectional
```

**Framing buffer drain** — after sending `Accept`, the `bufio.Reader` may hold bytes that arrived before the raw proxy phase. These must be forwarded to the local connection before bidirectional copy starts:

```go
// drain whatever bufio buffered after the framing exchange
if n := br.Buffered(); n > 0 {
    buf := make([]byte, n)
    br.Read(buf)
    localConn.Write(buf)
}
go io.Copy(localConn, remoteConn)
go io.Copy(remoteConn, localConn)
```

### CLI flag

```
-tunnel    Open a bore.pub tunnel and print the public URL
```

When `-tunnel` is set:
- Bore starts after the HTTP server is listening (port known)
- Public URL printed instead of localhost URL
- Bore stopped after `Wait()` returns

## UI Redesign

### Palette (Solarized Light)

```css
--sol-base3:  #fdf6e3   /* page background */
--sol-base2:  #eee8d5   /* card + input background */
--sol-base1:  #93a1a1   /* secondary / placeholder text */
--sol-base0:  #657b83   /* body text */
--sol-base00: #586e75   /* headings / strong text */
--sol-cyan:   #2aa198   /* primary accent: button, focus ring */
--sol-blue:   #268bd2   /* links */
--sol-yellow: #b58900   /* badge */
--sol-red:    #dc322f   /* errors */
--sol-green:  #859900   /* success */
```

### ASCII art header

```
██╗   ██╗ █████╗ ██╗   ██╗██╗  ████████╗
██║   ██║██╔══██╗██║   ██║██║  ╚══██╔══╝
██║   ██║███████║██║   ██║██║     ██║
╚██╗ ██╔╝██╔══██║██║   ██║██║     ██║
 ╚████╔╝ ██║  ██║╚██████╔╝███████╗██║
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝
```

Rendered in a `<pre>` with `background: linear-gradient(...)` across solarized accents (yellow → orange → cyan). Animated with `gradientDrift` (same as artfct — oscillates `background-position` over 10s).

### Component inventory

| Element | Treatment |
|---|---|
| Page bg | `sol-base3` (#fdf6e3) |
| Card | `sol-base2` bg, subtle border `sol-base1`, border-radius 12px |
| Badge `vault request` | `sol-yellow` text, `sol-base2` bg, monospace |
| Key name heading | `sol-base00`, blinking `▋` cursor (`.cursor-blink`) |
| Note text | `sol-base1`, 0.9rem |
| Password input | `sol-base2` bg, `sol-base00` text, `sol-base1` border → cyan on focus |
| Submit button | `sol-cyan` bg, `sol-base3` text, scale(0.97) on active |
| Success fragment | `sol-green` text, "saved!" message, `window.close()` after 2s |
| Error fragment | `sol-red` text |
| Expired page | Same Solarized Light treatment, red heading |

### Typography

Monospace stack throughout (no external font — self-contained binary):
```css
font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
```

## Testing

### Updated tests

- `TestEndToEndClaimFlow` — POST encrypted blob (base64 of iv ‖ aes-gcm ciphertext); assert `Wait()` returns correct blob; assert CLI decryption round-trip produces original value
- `TestBadTokenReturns404` — no change
- `TestTokenReuseBlocked` — no change

### New tests

- `TestDecryptionRoundTrip` — unit test: encrypt in Go (same AES-GCM), decrypt, assert plaintext matches
- `TestBoreTunnel` (smoke) — if `bore.pub` reachable, dial, get port > 0, stop cleanly; skip if offline

## File structure after changes

```
├── main.go
├── cmd/request.go          (key gen, --tunnel, decrypt, store)
├── server/
│   ├── server.go           (drop Store, add encryptedBlob, Wait → (bool,[]byte))
│   └── handlers.go         (WebCrypto JS, Solarized Light UI)
├── token/token.go          (unchanged)
├── store/store.go          (unchanged)
├── tunnel/
│   └── bore.go             (new — bore.pub client)
└── vault_test.go           (updated + new tests)
```

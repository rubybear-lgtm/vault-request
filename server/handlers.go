package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/rubybear-lgtm/vault-request/token"
)

// FormData is passed to the HTML template.
type FormData struct {
	Name  string
	Note  string
	Token string
}

type resultData struct {
	Success bool
	Name    string
	Error   string
}

// Template strings (no external files needed).
const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<script src="https://unpkg.com/htmx.org@2"></script>
<title>Secret: {{.Name}}</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    background: #1a1a2e;
    color: #e0e0e0;
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
  }
  .card {
    background: #16213e;
    border-radius: 12px;
    padding: 2rem;
    max-width: 440px;
    width: 100%;
    box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  }
  h1 { font-size: 1.5rem; margin-bottom: 0.5rem; color: #fff; }
  .note { color: #8892b0; font-size: 0.9rem; margin-bottom: 1.5rem; }
  label { display: block; margin-bottom: 0.5rem; font-size: 0.85rem; color: #a8b2d1; }
  input[type="password"] {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #233554;
    border-radius: 8px;
    background: #0a192f;
    color: #e0e0e0;
    font-size: 1rem;
    outline: none;
    transition: border-color 0.2s;
  }
  input[type="password"]:focus { border-color: #64ffda; }
  button {
    margin-top: 1rem;
    width: 100%;
    padding: 0.75rem;
    border: none;
    border-radius: 8px;
    background: #64ffda;
    color: #0a192f;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
  }
  button:hover { opacity: 0.9; }
  button:disabled { opacity: 0.5; cursor: not-allowed; }
  #result { margin-top: 1rem; }
  .success { text-align: center; color: #64ffda; }
  .success strong { display: block; font-size: 1.1rem; margin-bottom: 0.5rem; }
  .success small { color: #8892b0; }
  .error { text-align: center; color: #ff6b6b; }
  .spinner { display: inline-block; width: 16px; height: 16px; border: 2px solid #64ffda; border-top-color: transparent; border-radius: 50%; animation: spin 0.6s linear infinite; margin-right: 0.5rem; vertical-align: middle; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .badge { display: inline-block; background: #233554; padding: 0.25rem 0.5rem; border-radius: 4px; font-family: monospace; font-size: 0.8rem; margin-bottom: 1rem; color: #64ffda; }
</style>
</head>
<body>
<div class="card">
  <div class="badge">vault request</div>
  <h1>🔑 {{.Name}}</h1>
  {{if .Note}}<p class="note">{{.Note}}</p>{{end}}
  <form hx-post="/claim/{{.Token}}" hx-target="#result" hx-swap="innerHTML" hx-indicator="#submit-spinner">
    <label for="value">Enter the secret value</label>
    <input type="password" id="value" name="value" required autofocus spellcheck="false" autocomplete="off">
    <button type="submit" id="submit-btn">
      <span id="submit-spinner" class="spinner" style="display:none"></span>
      Submit
    </button>
  </form>
  <div id="result"></div>
</div>
</body>
</html>`

const successFragment = `<div class="success">
  <strong>✅ {{.Name}} saved!</strong>
  <small>This tab will close automatically.</small>
</div>
<script>setTimeout(() => window.close(), 2000)</script>`

const errorFragment = `<div class="error">
  <p>❌ {{.Error}}</p>
</div>`

// newRouter creates the http.Handler with all routes.
func (s *Server) newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/claim/", s.handleClaim)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	return mux
}

func (s *Server) handleClaim(w http.ResponseWriter, r *http.Request) {
	// Extract token from path: /claim/<token>
	tok := strings.TrimPrefix(r.URL.Path, "/claim/")
	if tok == "" {
		http.NotFound(w, r)
		return
	}

	s.mu.RLock()
	claimed := s.claimed
	valid := token.Validate(tok, s.token)
	s.mu.RUnlock()

	if !valid || claimed {
		s.renderExpired(w, tok)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.renderForm(w, tok)
	case http.MethodPost:
		s.handleSubmit(w, r, tok)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) renderForm(w http.ResponseWriter, tok string) {
	tmpl, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, FormData{
		Name:  s.secretName,
		Note:  s.secretNote,
		Token: tok,
	})
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request, tok string) {
	value := strings.TrimSpace(r.FormValue("value"))
	if value == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		t, _ := template.New("err").Parse(errorFragment)
		t.Execute(w, resultData{Error: "Value cannot be empty."})
		return
	}

	if err := s.store.Save(s.secretName, value); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		t, _ := template.New("err").Parse(errorFragment)
		t.Execute(w, resultData{Error: "Failed to save: " + err.Error()})
		return
	}

	// Mark token as claimed.
	s.mu.Lock()
	s.claimed = true
	s.mu.Unlock()

	// Signal the blocking command that we're done.
	close(s.done)

	// Send success response.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	t, _ := template.New("success").Parse(successFragment)
	t.Execute(w, resultData{Success: true, Name: s.secretName})
}

func (s *Server) renderExpired(w http.ResponseWriter, tok string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	html := `<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>body { font-family: sans-serif; background: #1a1a2e; color: #e0e0e0; display: flex; align-items: center; justify-content: center; min-height: 100vh; text-align: center; } .card { background: #16213e; padding: 2rem; border-radius: 12px; } h1 { color: #ff6b6b; }</style>
</head><body><div class="card"><h1>❌ Link expired or already used</h1><p>This secret request link is no longer valid.</p></div></body></html>`
	w.Write([]byte(html))
}

// jsonResult returns a JSON result for agent-friendly CLI output.
func jsonResult(success bool, name, note string) string {
	type output struct {
		Success bool   `json:"success"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	msg := "Secret provisioned."
	if !success {
		msg = "Secret request expired or failed."
	}
	b, _ := json.Marshal(output{
		Success: success,
		Name:    name,
		Message: msg,
	})
	return string(b)
}

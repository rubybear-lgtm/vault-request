package server

import (
	"encoding/base64"
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

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>vault — {{.Name}}</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{
    font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;
    background:#fdf6e3;color:#657b83;
    min-height:100vh;display:flex;align-items:center;justify-content:center;padding:1.5rem;
  }
  .wrap{max-width:480px;width:100%}
  .ascii-hero{
    font-size:.52rem;line-height:1.2;white-space:pre;display:block;overflow-x:auto;
    margin-bottom:1.5rem;
    background:linear-gradient(90deg,#b58900,#cb4b16,#2aa198,#268bd2);
    -webkit-background-clip:text;-webkit-text-fill-color:transparent;
    background-clip:text;background-size:200% auto;
    animation:drift 8s ease-in-out infinite;
  }
  @keyframes drift{0%{background-position:0% center}50%{background-position:100% center}100%{background-position:0% center}}
  .card{background:#eee8d5;border:1px solid #93a1a1;border-radius:8px;padding:1.75rem}
  .badge{
    display:inline-block;background:#fdf6e3;border:1px solid #93a1a1;
    color:#b58900;font-size:.75rem;padding:.2rem .5rem;border-radius:4px;margin-bottom:1rem;
  }
  h1{font-size:1.05rem;color:#586e75;margin-bottom:.35rem}
  h1::after{content:'▋';animation:blink 1s step-end infinite;margin-left:2px;color:#2aa198}
  @keyframes blink{0%,100%{opacity:1}50%{opacity:0}}
  .note{font-size:.85rem;color:#93a1a1;margin-bottom:1.25rem}
  .key-error{
    background:#fdf6e3;border:1px solid #dc322f;color:#dc322f;
    padding:.6rem .75rem;border-radius:4px;font-size:.82rem;margin-bottom:1rem;
  }
  label{display:block;font-size:.8rem;color:#93a1a1;margin-bottom:.4rem}
  input[type=password]{
    width:100%;padding:.65rem .75rem;border:1px solid #93a1a1;border-radius:4px;
    background:#fdf6e3;color:#586e75;font-family:inherit;font-size:.95rem;
    outline:none;transition:border-color .15s;
  }
  input[type=password]:focus{border-color:#2aa198}
  button{
    margin-top:.9rem;width:100%;padding:.65rem;border:none;border-radius:4px;
    background:#2aa198;color:#fdf6e3;font-family:inherit;font-size:.95rem;font-weight:600;
    cursor:pointer;transition:opacity .15s,transform .05s;
  }
  button:hover:not(:disabled){opacity:.88}
  button:active:not(:disabled){transform:scale(.97)}
  button:disabled{opacity:.45;cursor:not-allowed}
  #result{margin-top:.9rem;font-size:.9rem}
  .ok{color:#859900}.ok strong{display:block;margin-bottom:.25rem}
  .err{color:#dc322f}
</style>
</head>
<body>
<div class="wrap">
<pre class="ascii-hero">
&#x2588;&#x2588;&#x2557;   &#x2588;&#x2588;&#x2557; &#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2557;&#x2588;&#x2588;&#x2557;   &#x2588;&#x2588;&#x2557;&#x2588;&#x2588;&#x2557;  &#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2557;
&#x2588;&#x2588;&#x2551;   &#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2554;&#x2550;&#x2550;&#x2588;&#x2588;&#x2557;&#x2588;&#x2588;&#x2551;   &#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2551;  &#x255A;&#x2550;&#x2550;&#x2588;&#x2588;&#x2554;&#x2550;&#x2550;&#x255D;
&#x2588;&#x2588;&#x2551;   &#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2551;   &#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2551;     &#x2588;&#x2588;&#x2551;
&#x255A;&#x2588;&#x2588;&#x2557; &#x2588;&#x2588;&#x2554;&#x255D;&#x2588;&#x2588;&#x2554;&#x2550;&#x2550;&#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2551;   &#x2588;&#x2588;&#x2551;&#x2588;&#x2588;&#x2551;     &#x2588;&#x2588;&#x2551;
 &#x255A;&#x2588;&#x2588;&#x2588;&#x2588;&#x2554;&#x255D; &#x2588;&#x2588;&#x2551;  &#x2588;&#x2588;&#x2551;&#x255A;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2554;&#x255D;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2588;&#x2557;&#x2588;&#x2588;&#x2551;
  &#x255A;&#x2550;&#x2550;&#x2550;&#x255D;  &#x255A;&#x2550;&#x255D;  &#x255A;&#x2550;&#x255D; &#x255A;&#x2550;&#x2550;&#x2550;&#x2550;&#x2550;&#x255D; &#x255A;&#x2550;&#x2550;&#x2550;&#x2550;&#x2550;&#x255D;&#x255A;&#x2550;&#x255D;
</pre>
<div class="card">
  <span class="badge">vault request</span>
  <h1>{{.Name}}</h1>
  {{if .Note}}<p class="note">{{.Note}}</p>{{end}}
  <div id="key-error" class="key-error" style="display:none"></div>
  <div id="form-area">
    <label for="value">secret value</label>
    <input type="password" id="value" name="value" autofocus spellcheck="false" autocomplete="off">
    <button id="submit-btn" type="button" onclick="submitSecret()">submit</button>
  </div>
  <div id="result"></div>
</div>
</div>
<script>
let cryptoKey=null;
(async function init(){
  const m=location.hash.match(/^#k=([0-9a-f]{64})$/);
  if(!m){
    const el=document.getElementById('key-error');
    el.textContent='Missing encryption key — re-request this link from the agent.';
    el.style.display='block';
    document.getElementById('submit-btn').disabled=true;
    return;
  }
  if(!window.crypto||!window.crypto.subtle){
    const el=document.getElementById('key-error');
    el.textContent='WebCrypto unavailable — use a modern browser over HTTPS or localhost.';
    el.style.display='block';
    document.getElementById('submit-btn').disabled=true;
    return;
  }
  const bytes=new Uint8Array(m[1].match(/.{2}/g).map(b=>parseInt(b,16)));
  cryptoKey=await crypto.subtle.importKey('raw',bytes,{name:'AES-GCM'},false,['encrypt']);
})();
async function submitSecret(){
  const val=document.getElementById('value').value.trim();
  if(!val){showErr('Value cannot be empty.');return;}
  if(!cryptoKey)return;
  const iv=crypto.getRandomValues(new Uint8Array(12));
  const ct=await crypto.subtle.encrypt({name:'AES-GCM',iv},cryptoKey,new TextEncoder().encode(val));
  const buf=new Uint8Array(12+ct.byteLength);
  buf.set(iv);buf.set(new Uint8Array(ct),12);
  let bin='';buf.forEach(b=>bin+=String.fromCharCode(b));
  const b64=btoa(bin);
  const btn=document.getElementById('submit-btn');
  btn.disabled=true;btn.textContent='submitting...';
  try{
    const r=await fetch('/claim/{{.Token}}',{
      method:'POST',
      headers:{'Content-Type':'application/x-www-form-urlencoded'},
      body:'value='+encodeURIComponent(b64),
    });
    document.getElementById('result').innerHTML=await r.text();
    if(r.ok){document.getElementById('form-area').style.display='none';}
    else{btn.disabled=false;btn.textContent='submit';}
  }catch(e){
    showErr('Network error: '+e.message);
    btn.disabled=false;btn.textContent='submit';
  }
}
function showErr(msg){
  document.getElementById('result').innerHTML='<p class="err">&#10060; '+msg+'</p>';
}
</script>
</body>
</html>`

const successFragment = `<div class="ok">
  <strong>&#10003; {{.Name}} saved.</strong>
  <span>You can close this tab.</span>
</div>
<script>setTimeout(()=>window.close(),2000)</script>`

const errorFragment = `<p class="err">&#10060; {{.Error}}</p>`

func (s *Server) newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/claim/", s.handleClaim)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	return mux
}

func (s *Server) handleClaim(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimPrefix(r.URL.Path, "/claim/")
	if tok == "" {
		http.NotFound(w, r)
		return
	}

	s.mu.RLock()
	submitted := s.submitted
	valid := token.Validate(tok, s.token)
	s.mu.RUnlock()

	if !valid || submitted {
		s.renderExpired(w)
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
	tmpl.Execute(w, FormData{Name: s.secretName, Note: s.secretNote, Token: tok})
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request, tok string) {
	raw := strings.TrimSpace(r.FormValue("value"))
	if raw == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		t, _ := template.New("e").Parse(errorFragment)
		t.Execute(w, resultData{Error: "Value cannot be empty."})
		return
	}

	blob, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(blob) < 13 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		t, _ := template.New("e").Parse(errorFragment)
		t.Execute(w, resultData{Error: "Invalid encrypted value."})
		return
	}

	s.setEncryptedBlob(blob)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, _ := template.New("s").Parse(successFragment)
	t.Execute(w, resultData{Success: true, Name: s.secretName})
}

func (s *Server) renderExpired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><style>
body{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;background:#fdf6e3;color:#657b83;display:flex;align-items:center;justify-content:center;min-height:100vh;text-align:center}
.card{background:#eee8d5;border:1px solid #93a1a1;padding:2rem;border-radius:8px;max-width:360px}
h1{color:#dc322f;font-size:1rem;margin-bottom:.5rem}p{color:#93a1a1;font-size:.85rem}
</style></head><body>
<div class="card"><h1>&#10060; link expired</h1><p>This secret request link is no longer valid.</p></div>
</body></html>`))
}

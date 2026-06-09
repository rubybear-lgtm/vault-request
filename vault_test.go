package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rubybear-lgtm/vault-request/server"
	"github.com/rubybear-lgtm/vault-request/store"
)

func TestEndToEndClaimFlow(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	s, err := store.NewEnvStore(envPath)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := server.Start(server.Config{
		Store:      s,
		SecretName: "TEST_KEY",
		Note:       "integration test key",
		TTL:        2 * time.Minute,
		Port:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	url := srv.URL()
	t.Logf("Server URL: %s", url)

	if !strings.Contains(url, "/claim/") {
		t.Fatalf("expected /claim/ in URL, got %s", url)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. GET form page
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET form: expected 200, got %d", resp.StatusCode)
	}
	body := readAll(t, resp)
	resp.Body.Close()
	if !strings.Contains(body, "TEST_KEY") {
		t.Fatal("form does not contain secret name")
	}
	if !strings.Contains(body, "htmx.org") {
		t.Fatal("form does not include htmx")
	}
	t.Log("✓ GET form returns 200 with htmx")

	// 2. POST empty → 400
	resp, err = client.Post(url, "application/x-www-form-urlencoded", strings.NewReader("value="))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST empty: expected 400, got %d", resp.StatusCode)
	}
	t.Log("✓ POST empty value returns 400")

	// 3. POST valid → 200
	resp, err = client.Post(url, "application/x-www-form-urlencoded", strings.NewReader("value=sk-test-abc"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST valid: expected 200, got %d", resp.StatusCode)
	}
	successBody := readAll(t, resp)
	resp.Body.Close()
	if !strings.Contains(successBody, "saved") {
		t.Fatal("success response missing 'saved'")
	}
	t.Log("✓ POST valid value returns 200 with success")

	// 4. Wait for completion
	if !srv.Wait() {
		t.Fatal("expected Wait to return true")
	}
	t.Log("✓ Wait reports claimed")

	// 5. Verify env file
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `TEST_KEY="sk-test-abc"`) {
		t.Fatalf("env file missing expected value. Got:\n%s", content)
	}
	t.Log("✓ .env file contains correct value")
}

func TestBadTokenReturns404(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	s, _ := store.NewEnvStore(envPath)
	srv, _ := server.Start(server.Config{
		Store:      s,
		SecretName: "BAD_TOKEN_TEST",
		TTL:        2 * time.Minute,
		Port:       0,
	})
	defer srv.Stop()

	badURL := fmt.Sprintf("http://127.0.0.1:%d/claim/invalidtoken123", srv.Port())
	client := &http.Client{Timeout: 3 * time.Second}

	// GET with bad token
	resp, err := client.Get(badURL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for GET bad token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// POST with bad token
	resp, err = client.Post(badURL, "application/x-www-form-urlencoded", strings.NewReader("value=test"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for POST bad token, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✓ Bad token returns 404 for GET and POST")
}

func TestTokenReuseBlocked(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	s, _ := store.NewEnvStore(envPath)
	srv, _ := server.Start(server.Config{
		Store:      s,
		SecretName: "REUSE_TEST",
		TTL:        2 * time.Minute,
		Port:       0,
	})
	defer srv.Stop()

	url := srv.URL()
	client := &http.Client{Timeout: 3 * time.Second}

	// First claim
	resp, err := client.Post(url, "application/x-www-form-urlencoded", strings.NewReader("value=first"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first claim: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	srv.Wait()

	// Second attempt should 404 (token already claimed)
	resp, err = client.Post(url, "application/x-www-form-urlencoded", strings.NewReader("value=second"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("reuse: expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✓ Token reuse returns 404")
}

func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

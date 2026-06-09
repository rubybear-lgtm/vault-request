package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rubybear-lgtm/vault-request/store"
	"github.com/rubybear-lgtm/vault-request/token"
)

// Server manages the one-time HTTP server lifecycle.
type Server struct {
	secretName string
	secretNote string
	token      string
	claimed    bool
	store      store.Store

	mu     sync.RWMutex
	done   chan struct{}
	http   *http.Server
	port   int
}

// Config holds server creation parameters.
type Config struct {
	Store      store.Store
	SecretName string
	Note       string
	TTL        time.Duration
	Port       int // 0 = random
	ListenAddr string // default "127.0.0.1"
}

// Start creates and starts the HTTP server, returning once the server is
// listening. The server runs in the background; call Wait() to block until
// completion.
func Start(cfg Config) (*Server, error) {
	tok, err := token.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	listenAddr := cfg.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	s := &Server{
		secretName: cfg.SecretName,
		secretNote: cfg.Note,
		token:      tok,
		store:      cfg.Store,
		done:       make(chan struct{}),
	}

	s.http = &http.Server{
		Handler:           s.newRouter(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenAddr, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	s.port = listener.Addr().(*net.TCPAddr).Port

	go func() {
		s.http.Serve(listener)
	}()

	// Auto-shutdown on TTL timeout. On claim, we only close the done
	// channel; the server stays alive until the TTL or Stop().
	go func() {
		select {
		case <-time.After(cfg.TTL):
			s.mu.Lock()
			if !s.claimed {
				s.claimed = true
				close(s.done)
			}
			s.mu.Unlock()
			s.shutdown()
		case <-s.done:
			// claimed — don't shutdown immediately; allow
			// re-claim detection and graceful HTTP completion.
		}
	}()

	return s, nil
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Token returns the one-time claim token.
func (s *Server) Token() string {
	return s.token
}

// URL returns the full claim URL for the user.
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/claim/%s", s.port, s.token)
}

// Done returns a channel that closes when the secret is claimed or the TTL
// expires.
func (s *Server) Done() <-chan struct{} {
	return s.done
}

// Wait blocks until the secret is claimed or TTL expires.
// Returns true if the secret was successfully saved.
func (s *Server) Wait() bool {
	<-s.done
	s.mu.RLock()
	claimed := s.claimed
	s.mu.RUnlock()
	return claimed
}

// Stop forces the server to shut down immediately.
// Safe to call multiple times. Does not affect claimed state or done channel.
func (s *Server) Stop() {
	s.shutdown()
}

func (s *Server) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.http.Shutdown(ctx)
}

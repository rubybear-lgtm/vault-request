package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rubybear-lgtm/vault-request/token"
)

// Server manages the one-time HTTP server lifecycle.
type Server struct {
	secretName    string
	secretNote    string
	token         string
	submitted     bool
	encryptedBlob []byte

	mu       sync.RWMutex
	done     chan struct{}
	doneOnce sync.Once
	http     *http.Server
	port     int
}

// Config holds server creation parameters.
type Config struct {
	SecretName string
	Note       string
	TTL        time.Duration
	Port       int
	ListenAddr string
}

// Start creates and starts the HTTP server, returning once it is listening.
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

	go s.http.Serve(listener)

	go func() {
		select {
		case <-time.After(cfg.TTL):
			s.signalDone()
			s.shutdown()
		case <-s.done:
		}
	}()

	return s, nil
}

func (s *Server) Port() int     { return s.port }
func (s *Server) Token() string { return s.token }

func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/claim/%s", s.port, s.token)
}

func (s *Server) Done() <-chan struct{} { return s.done }

// Wait blocks until submitted or TTL expires.
// Returns (true, encryptedBlob) on success, (false, nil) on timeout.
func (s *Server) Wait() (bool, []byte) {
	<-s.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.submitted, s.encryptedBlob
}

func (s *Server) Stop() { s.shutdown() }

func (s *Server) setEncryptedBlob(blob []byte) {
	s.mu.Lock()
	s.submitted = true
	s.encryptedBlob = blob
	s.mu.Unlock()
	s.signalDone()
}

func (s *Server) signalDone() {
	s.doneOnce.Do(func() { close(s.done) })
}

func (s *Server) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s.http.Shutdown(ctx)
}

package cmd

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rubybear-lgtm/vault-request/server"
	"github.com/rubybear-lgtm/vault-request/store"
	"github.com/rubybear-lgtm/vault-request/token"
	"github.com/rubybear-lgtm/vault-request/tunnel"
	"golang.org/x/crypto/nacl/secretbox"
)

// RequestConfig holds parsed CLI flags for the "request" command.
type RequestConfig struct {
	SecretName string
	Note       string
	Port       int
	TTLMinutes int
	ListenAddr string
	OutFile    string
	JSONOutput bool
	Tunnel     bool
}

// RequestOutput is the JSON-serializable result printed after completion.
type RequestOutput struct {
	Success bool   `json:"success"`
	Name    string `json:"name"`
	Message string `json:"message"`
	URL     string `json:"url"`
	Port    int    `json:"port"`
}

// RunRequest executes the "vault request <name>" subcommand.
func RunRequest(args []string) error {
	cfg, err := parseRequestFlags(args)
	if err != nil {
		return err
	}

	s, err := store.NewEnvStore(cfg.OutFile)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}

	// The encryption key lives only in the URL fragment — never sent to server or relay.
	encKeyHex, err := token.Generate()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	keyBytes, _ := hex.DecodeString(encKeyHex)

	srv, err := server.Start(server.Config{
		SecretName: cfg.SecretName,
		Note:       cfg.Note,
		TTL:        time.Duration(cfg.TTLMinutes) * time.Minute,
		Port:       cfg.Port,
		ListenAddr: cfg.ListenAddr,
	})
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/claim/%s#k=%s", srv.Port(), srv.Token(), encKeyHex)

	if cfg.Tunnel {
		tun, err := tunnel.Start(srv.Port())
		if err != nil {
			return fmt.Errorf("bore tunnel: %w", err)
		}
		defer tun.Stop()
		url = fmt.Sprintf("http://bore.pub:%d/claim/%s#k=%s", tun.RemotePort(), srv.Token(), encKeyHex)
	}

	if cfg.JSONOutput {
		out := RequestOutput{
			Success: true,
			Name:    cfg.SecretName,
			Message: fmt.Sprintf("Secret request link (one-time, expires in %d min)", cfg.TTLMinutes),
			URL:     url,
			Port:    srv.Port(),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		tunnelNote := ""
		if cfg.Tunnel {
			tunnelNote = " via bore.pub"
		}
		fmt.Println()
		fmt.Printf("  🔗  Secret request link (one-time, E2E encrypted%s)\n", tunnelNote)
		fmt.Println()
		fmt.Printf("     %s\n", url)
		fmt.Println()
		fmt.Printf("     Name:   %s\n", cfg.SecretName)
		if cfg.Note != "" {
			fmt.Printf("     Note:   %s\n", cfg.Note)
		}
		fmt.Printf("     Expiry: %d minutes\n", cfg.TTLMinutes)
		fmt.Println()
		fmt.Println("     Waiting for user to submit...")
		fmt.Println()
	}

	ok, encBlob := srv.Wait()

	if ok {
		plain, err := decryptBlob(keyBytes, encBlob)
		if err != nil {
			return fmt.Errorf("decrypt: %w", err)
		}
		if err := s.Save(cfg.SecretName, plain); err != nil {
			return fmt.Errorf("save: %w", err)
		}
	}

	if cfg.JSONOutput {
		out := RequestOutput{
			Success: ok,
			Name:    cfg.SecretName,
			Port:    srv.Port(),
		}
		if ok {
			out.Message = fmt.Sprintf("Secret '%s' provisioned successfully.", cfg.SecretName)
		} else {
			out.Message = "Request timed out or failed."
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		if ok {
			fmt.Println()
			fmt.Printf("  ✅  Secret '%s' saved to %s\n", cfg.SecretName, cfg.OutFile)
			fmt.Println()
		} else {
			fmt.Println()
			fmt.Println("  ❌  Request timed out. No secret was saved.")
			fmt.Println()
			os.Exit(1)
		}
	}

	return nil
}

// decryptBlob decrypts a nacl/secretbox payload (nonce[24] || mac+ciphertext).
func decryptBlob(key, blob []byte) (string, error) {
	if len(blob) < 24+secretbox.Overhead {
		return "", fmt.Errorf("blob too short (%d bytes)", len(blob))
	}
	var nonce [24]byte
	copy(nonce[:], blob[:24])
	var k [32]byte
	copy(k[:], key)
	plain, ok := secretbox.Open(nil, blob[24:], &nonce, &k)
	if !ok {
		return "", fmt.Errorf("decryption failed (tampered payload?)")
	}
	return string(plain), nil
}

func parseRequestFlags(args []string) (*RequestConfig, error) {
	var secretName string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if isValueFlag(arg) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else if secretName == "" {
			secretName = arg
		} else {
			flagArgs = append(flagArgs, arg)
		}
	}

	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	cfg := &RequestConfig{}
	fs.StringVar(&cfg.Note, "note", "", "Human-readable description shown on the form")
	fs.IntVar(&cfg.Port, "port", 0, "Port to bind (default: random available)")
	fs.IntVar(&cfg.TTLMinutes, "ttl", 30, "Minutes until the link expires")
	fs.StringVar(&cfg.ListenAddr, "listen-addr", "127.0.0.1", "Address to listen on")
	fs.StringVar(&cfg.OutFile, "out", ".env", "Output .env file path")
	fs.BoolVar(&cfg.JSONOutput, "json", false, "Output as JSON for agent parsing")
	fs.BoolVar(&cfg.Tunnel, "tunnel", false, "Open a bore.pub tunnel and print the public URL")

	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}
	if secretName == "" {
		return nil, fmt.Errorf("usage: vault request <secret-name> [flags]\n\nRun 'vault --help' for available flags.")
	}
	cfg.SecretName = secretName
	return cfg, nil
}

func isValueFlag(arg string) bool {
	name := strings.TrimLeft(arg, "-")
	switch name {
	case "note", "port", "ttl", "listen-addr", "out":
		return true
	default:
		return false
	}
}

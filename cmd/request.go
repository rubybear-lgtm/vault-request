package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rubybear-lgtm/vault-request/server"
	"github.com/rubybear-lgtm/vault-request/store"
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

	srv, err := server.Start(server.Config{
		Store:      s,
		SecretName: cfg.SecretName,
		Note:       cfg.Note,
		TTL:        time.Duration(cfg.TTLMinutes) * time.Minute,
		Port:       cfg.Port,
		ListenAddr: cfg.ListenAddr,
	})
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	url := srv.URL()

	// Print the link prominently.
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
		fmt.Println()
		fmt.Println("  🔗  Secret request link (one-time claim)")
		fmt.Println()
		fmt.Printf("     %s\n", url)
		fmt.Println()
		fmt.Printf("     Name:   %s\n", cfg.SecretName)
		if cfg.Note != "" {
			fmt.Printf("     Note:   %s\n", cfg.Note)
		}
		fmt.Printf("     Expiry: %d minutes\n", cfg.TTLMinutes)
		fmt.Println()
		fmt.Println("     Share this link with the user to fill in the value.")
		fmt.Println("     Waiting for user to submit...")
		fmt.Println()
	}

	// Block until claimed or timeout.
	success := srv.Wait()

	if cfg.JSONOutput {
		out := RequestOutput{
			Success: success,
			Name:    cfg.SecretName,
			Port:    srv.Port(),
		}
		if success {
			out.Message = fmt.Sprintf("Secret '%s' provisioned successfully.", cfg.SecretName)
		} else {
			out.Message = "Request timed out or failed."
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		if success {
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

func parseRequestFlags(args []string) (*RequestConfig, error) {
	// Scan args to find the first non-flag argument (the secret name).
	// We extract it so flags can appear before or after the name.
	var secretName string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// If this flag takes a value and the next arg isn't a flag, include it.
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

	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}

	if secretName == "" {
		return nil, fmt.Errorf("usage: vault request <secret-name> [flags]\n\nRun 'vault --help' for available flags.")
	}

	cfg.SecretName = secretName
	return cfg, nil
}

// isValueFlag returns true if the flag name takes a value argument.
func isValueFlag(arg string) bool {
	name := strings.TrimLeft(arg, "-")
	switch name {
	case "note", "port", "ttl", "listen-addr", "out":
		return true
	default:
		return false
	}
}

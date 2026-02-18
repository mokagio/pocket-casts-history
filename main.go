// pocket-casts-history extracts your Pocket Casts listening history via the
// unofficial API and writes it to a local JSON file.
//
// On first run, it creates a config file at ~/.config/pocket-casts-history/config.yaml
// and prompts for settings. Credentials are read from the macOS Keychain.
package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		// In Go, it's conventional to print errors to stderr and exit with
		// a non-zero status. We use a separate run() function so main() can
		// handle the exit cleanly — os.Exit doesn't run deferred functions.
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run contains the actual program logic, returning an error on failure.
//
// Separating logic from main() is a common Go pattern. It lets us use
// normal error returns instead of os.Exit, making the flow easier to follow
// and test.
func run() error {
	// Step 1: Load or create config.
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}

	// Step 2: Read credentials from Keychain.
	fmt.Println("Reading credentials from Keychain...")
	creds, err := ReadCredentials(ExecRunner{}, cfg.KeychainItem)
	if err != nil {
		return fmt.Errorf("reading credentials: %w\n\nHave you stored them? Run:\n  security add-generic-password -a \"your@email.com\" -s %q -w", err, cfg.KeychainItem)
	}

	// Step 3: Login to Pocket Casts.
	fmt.Println("Logging in to Pocket Casts...")
	client := NewClient()
	token, err := client.Login(creds.Email, creds.Password)
	if err != nil {
		return fmt.Errorf("logging in: %w", err)
	}

	// Step 4: Fetch listening history.
	fmt.Println("Fetching listening history...")
	episodes, err := client.FetchHistory(token)
	if err != nil {
		return fmt.Errorf("fetching history: %w", err)
	}

	// Step 5: Write output file.
	outputDir, err := ExpandPath(cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("expanding output path: %w", err)
	}

	output := NewHistoryOutput(time.Now(), episodes)
	if err := WriteHistory(outputDir, output); err != nil {
		return fmt.Errorf("writing history: %w", err)
	}

	fmt.Printf("Wrote %d episodes to %s/history.json\n", len(episodes), outputDir)
	return nil
}

// loadOrCreateConfig loads the config from disk, or runs the first-run setup
// if no config file exists.
func loadOrCreateConfig() (Config, error) {
	cfgPath, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	cfg, err := LoadConfig(cfgPath)
	if err == nil {
		return cfg, nil
	}

	// errors.Is unwraps the error chain to check for os.ErrNotExist.
	// os.IsNotExist only checks the top-level error and won't match wrapped
	// errors (like the one from LoadConfig's fmt.Errorf with %w).
	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	// First run — prompt for config.
	fmt.Println("No config file found. Let's set things up.")
	cfg = PromptForConfig(os.Stdin, os.Stdout)

	if err := SaveConfig(cfgPath, cfg); err != nil {
		return Config{}, fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Config saved to %s\n\n", cfgPath)

	PrintKeychainInstructions(os.Stdout, cfg.KeychainItem)
	return cfg, nil
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the tool's persistent configuration.
//
// The struct tags tell the yaml package how to map Go fields to YAML keys.
// Using `yaml:"..."` is the idiomatic way to control serialization in Go.
type Config struct {
	OutputDir    string `yaml:"output_dir"`
	KeychainItem string `yaml:"keychain_item"`
}

// DefaultConfig returns a Config with sensible defaults.
//
// In Go, constructor-like functions are conventionally named NewX or DefaultX.
// They return a value (not a pointer) when the struct is small and has no
// internal state requiring heap allocation.
func DefaultConfig() Config {
	return Config{
		OutputDir:    "~/.local/share/pocket-casts-history",
		KeychainItem: "pocket-casts-api-login",
	}
}

// ConfigPath returns the full path to the config file.
//
// It follows XDG conventions: config goes under ~/.config/<app>/.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pocket-casts-history", "config.yaml"), nil
}

// ExpandPath replaces a leading "~" with the user's home directory.
//
// This is a common pattern in CLI tools — the shell doesn't expand "~" inside
// quoted strings or config files, so the application must handle it.
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	// Replace only the leading "~", preserving the rest of the path.
	return filepath.Join(home, path[1:]), nil
}

// LoadConfig reads the config file from disk.
//
// In Go, returning (value, error) is the standard pattern for fallible
// operations. The caller checks err != nil before using the value.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	// yaml.Unmarshal parses YAML bytes into the struct, guided by the
	// `yaml:"..."` tags we defined on Config.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes the config to disk, creating parent directories as needed.
//
// The os.MkdirAll call is idempotent — safe to call even if the directory
// already exists. 0o755 sets standard directory permissions (rwxr-xr-x).
func SaveConfig(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// 0o644 = owner read/write, group and others read-only — standard for
	// config files that don't contain secrets.
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// PromptForConfig interactively asks the user to configure the tool.
//
// It reads from r and writes prompts to w, making it testable — we can pass
// strings.NewReader and bytes.Buffer instead of os.Stdin/os.Stdout.
// This pattern of accepting io.Reader/io.Writer is idiomatic Go for I/O code
// that needs to be tested.
func PromptForConfig(r io.Reader, w io.Writer) Config {
	scanner := bufio.NewScanner(r)
	defaults := DefaultConfig()

	fmt.Fprintf(w, "Output directory [%s]: ", defaults.OutputDir)
	outputDir := defaults.OutputDir
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			outputDir = input
		}
	}

	fmt.Fprintf(w, "Keychain item name [%s]: ", defaults.KeychainItem)
	keychainItem := defaults.KeychainItem
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			keychainItem = input
		}
	}

	return Config{
		OutputDir:    outputDir,
		KeychainItem: keychainItem,
	}
}

// PrintKeychainInstructions tells the user how to store credentials.
func PrintKeychainInstructions(w io.Writer, keychainItem string) {
	fmt.Fprintf(w, "\nStore your Pocket Casts credentials in the macOS Keychain:\n\n")
	fmt.Fprintf(w, "  security add-generic-password -a \"your@email.com\" -s %q -w\n\n", keychainItem)
	fmt.Fprintf(w, "You'll be prompted for your password.\n")
}

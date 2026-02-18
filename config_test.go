package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExpandPath verifies tilde expansion for various inputs.
//
// This is a table-driven test — the idiomatic Go pattern for testing a single
// function with many inputs. Each "test case" is a struct in a slice; the test
// loops over them and calls t.Run for each, giving clear output on failure.
func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "tilde with subpath",
			input: "~/.config/test",
			want:  filepath.Join(home, ".config", "test"),
		},
		{
			name:  "tilde alone",
			input: "~",
			want:  home,
		},
		{
			name:  "absolute path unchanged",
			input: "/tmp/foo",
			want:  "/tmp/foo",
		},
		{
			name:  "relative path unchanged",
			input: "relative/path",
			want:  "relative/path",
		},
	}

	for _, tt := range tests {
		// t.Run creates a subtest — each case gets its own name in output.
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExpandPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestConfigRoundTrip verifies that saving and loading a config produces the
// same values.
//
// Using t.TempDir() gives us a fresh temporary directory that Go's test
// framework cleans up automatically after the test finishes.
func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subdir", "config.yaml")

	original := Config{
		OutputDir:    "~/my-output",
		KeychainItem: "my-keychain-item",
	}

	if err := SaveConfig(path, original); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded != original {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", loaded, original)
	}
}

// TestLoadConfigDefaults verifies that a minimal YAML file leaves missing
// fields at their zero values. (The caller should apply defaults separately.)
func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestPromptForConfig checks the interactive prompt with both default and
// custom input.
func TestPromptForConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Config
	}{
		{
			name:  "accept defaults (empty input)",
			input: "\n\n",
			want:  DefaultConfig(),
		},
		{
			name:  "custom values",
			input: "/tmp/my-output\nmy-custom-keychain\n",
			want: Config{
				OutputDir:    "/tmp/my-output",
				KeychainItem: "my-custom-keychain",
			},
		},
		{
			name:  "custom output, default keychain",
			input: "/tmp/custom\n\n",
			want: Config{
				OutputDir:    "/tmp/custom",
				KeychainItem: "pocket-casts-api-login",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			// bytes.Buffer implements io.Writer — we capture prompt output
			// here but don't need to assert on it.
			var w bytes.Buffer

			got := PromptForConfig(r, &w)
			if got != tt.want {
				t.Errorf("PromptForConfig:\n  got:  %+v\n  want: %+v", got, tt.want)
			}
		})
	}
}

// TestSaveConfigCreatesDirectories checks that SaveConfig creates parent dirs.
func TestSaveConfigCreatesDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "c", "config.yaml")

	err := SaveConfig(path, DefaultConfig())
	if err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Verify the file exists by reading it back.
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

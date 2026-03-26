package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Credentials holds the email and password for API authentication.
type Credentials struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

// CommandRunner abstracts shell command execution.
//
// In Go, interfaces are defined by the consumer, not the producer. This is
// the opposite of languages like Java. We define this small interface here
// so we can swap in a fake for testing — no real Keychain calls in unit tests.
type CommandRunner interface {
	// Run executes a command and returns its combined stdout+stderr output.
	Run(name string, args ...string) ([]byte, error)
}

// ExecRunner implements CommandRunner using os/exec — the real thing.
//
// An empty struct costs zero bytes — it's a common Go pattern for types
// that only carry methods, not data.
type ExecRunner struct{}

// Run executes the command and returns its combined output.
func (ExecRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// acctRegexp matches the "acct" attribute in `security find-generic-password -g` output.
//
// The output looks like:
//
//	keychain: "/Users/user/Library/Keychains/login.keychain-db"
//	...
//	    "acct"<blob>="user@example.com"
//	...
//	password: "s3cret"
//
// We use regexp.MustCompile at package level — it compiles once at init time
// and panics if the pattern is invalid (which is fine for hard-coded patterns).
var acctRegexp = regexp.MustCompile(`"acct"<blob>="([^"]*)"`)

// passwordRegexp matches the password line in `security -g` output.
// The password may be quoted ("secret") or hex (0x...).
var passwordRegexp = regexp.MustCompile(`password: "([^"]*)"`)

// ReadCredentials retrieves email and password from the macOS Keychain.
//
// It calls `security find-generic-password -s <service> -g` which outputs
// both the account (email) and the password to stderr. The -g flag is what
// triggers password output.
//
// The login keychain path is passed explicitly so the command works in
// restricted environments (e.g. cron) where the default search list
// doesn't include the login keychain.
func ReadCredentials(runner CommandRunner, keychainItem string) (Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Credentials{}, fmt.Errorf("resolving home directory: %w", err)
	}
	keychainPath := filepath.Join(home, "Library", "Keychains", "login.keychain-db")

	output, err := runner.Run("security", "find-generic-password", "-s", keychainItem, "-g", keychainPath)
	if err != nil {
		return Credentials{}, fmt.Errorf("reading keychain item %q: %w\noutput: %s", keychainItem, err, output)
	}

	creds, err := ParseSecurityOutput(string(output))
	if err != nil {
		return Credentials{}, fmt.Errorf("parsing keychain output for %q: %w", keychainItem, err)
	}
	return creds, nil
}

// ParseSecurityOutput extracts email and password from the raw output of
// `security find-generic-password -g`.
//
// Exported so we can test it directly with captured output — no Keychain needed.
func ParseSecurityOutput(output string) (Credentials, error) {
	acctMatch := acctRegexp.FindStringSubmatch(output)
	if acctMatch == nil {
		return Credentials{}, fmt.Errorf("could not find account (email) in security output")
	}

	pwMatch := passwordRegexp.FindStringSubmatch(output)
	if pwMatch == nil {
		return Credentials{}, fmt.Errorf("could not find password in security output")
	}

	email := strings.TrimSpace(acctMatch[1])
	password := strings.TrimSpace(pwMatch[1])

	if email == "" {
		return Credentials{}, fmt.Errorf("account (email) is empty")
	}
	if password == "" {
		return Credentials{}, fmt.Errorf("password is empty")
	}

	return Credentials{Email: email, Password: password}, nil
}

// CredentialsFilePath returns the path to the file-based credentials store.
//
// It lives alongside the config file under ~/.config/pocket-casts-history/.
func CredentialsFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pocket-casts-history", "credentials.yaml"), nil
}

// ReadCredentialsFile reads email and password from a YAML file.
//
// This is the preferred credential source for unattended environments
// (e.g. cron) where the macOS Keychain is inaccessible.
// The file should have 0600 permissions.
func ReadCredentialsFile(path string) (Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, fmt.Errorf("reading credentials file: %w", err)
	}

	var creds Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return Credentials{}, fmt.Errorf("parsing credentials file: %w", err)
	}

	if creds.Email == "" {
		return Credentials{}, fmt.Errorf("email is empty in credentials file")
	}
	if creds.Password == "" {
		return Credentials{}, fmt.Errorf("password is empty in credentials file")
	}

	return creds, nil
}

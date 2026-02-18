package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Credentials holds the email and password retrieved from the Keychain.
type Credentials struct {
	Email    string
	Password string
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
func ReadCredentials(runner CommandRunner, keychainItem string) (Credentials, error) {
	output, err := runner.Run("security", "find-generic-password", "-s", keychainItem, "-g")
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

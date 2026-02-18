package main

import (
	"fmt"
	"testing"
)

// fakeRunner is a test double implementing CommandRunner.
//
// In Go, you typically implement interfaces inline for tests rather than
// using a mocking framework. This keeps tests explicit and dependency-free.
type fakeRunner struct {
	output []byte
	err    error
}

func (f fakeRunner) Run(name string, args ...string) ([]byte, error) {
	return f.output, f.err
}

// sampleSecurityOutput mimics the real output of:
//
//	security find-generic-password -s "pocket-casts-api-login" -g
//
// Captured from an actual macOS system (with redacted values).
const sampleSecurityOutput = `keychain: "/Users/user/Library/Keychains/login.keychain-db"
version: 512
class: "genp"
attributes:
    0x00000007 <blob>="pocket-casts-api-login"
    "acct"<blob>="user@example.com"
    "cdat"<timedate>=0x32303236303231373132303030305A00  "20260217120000Z\000"
    "crtr"<uint32>=<NULL>
    "desc"<blob>=<NULL>
    "gena"<blob>=<NULL>
    "icmt"<blob>=<NULL>
    "labl"<blob>="pocket-casts-api-login"
    "mdat"<timedate>=0x32303236303231373132303030305A00  "20260217120000Z\000"
    "nega"<sint32>=<NULL>
    "prot"<blob>=<NULL>
    "scrp"<sint32>=<NULL>
    "svce"<blob>="pocket-casts-api-login"
    "type"<uint32>=<NULL>
password: "s3cret-passw0rd"`

// TestParseSecurityOutput verifies parsing of `security` command output.
func TestParseSecurityOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    Credentials
		wantErr bool
	}{
		{
			name:   "valid output",
			output: sampleSecurityOutput,
			want: Credentials{
				Email:    "user@example.com",
				Password: "s3cret-passw0rd",
			},
		},
		{
			name:    "missing account",
			output:  `password: "secret"`,
			wantErr: true,
		},
		{
			name:    "missing password",
			output:  `"acct"<blob>="user@example.com"`,
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name: "empty account value",
			output: `"acct"<blob>=""
password: "secret"`,
			wantErr: true,
		},
		{
			name: "empty password value",
			output: `"acct"<blob>="user@example.com"
password: ""`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSecurityOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSecurityOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSecurityOutput():\n  got:  %+v\n  want: %+v", got, tt.want)
			}
		})
	}
}

// TestReadCredentials verifies the full flow using a fake runner.
func TestReadCredentials(t *testing.T) {
	tests := []struct {
		name    string
		runner  CommandRunner
		want    Credentials
		wantErr bool
	}{
		{
			name: "successful read",
			runner: fakeRunner{
				output: []byte(sampleSecurityOutput),
			},
			want: Credentials{
				Email:    "user@example.com",
				Password: "s3cret-passw0rd",
			},
		},
		{
			name: "command fails",
			runner: fakeRunner{
				output: []byte("security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain."),
				err:    fmt.Errorf("exit status 44"),
			},
			wantErr: true,
		},
		{
			name: "unparseable output",
			runner: fakeRunner{
				output: []byte("garbage output"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadCredentials(tt.runner, "test-item")
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadCredentials():\n  got:  %+v\n  want: %+v", got, tt.want)
			}
		})
	}
}

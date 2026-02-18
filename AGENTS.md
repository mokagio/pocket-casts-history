# pocket-casts-history

Go CLI tool that extracts Pocket Casts listening history via the unofficial API and writes it to a JSON file.

## Purpose

This is both a utility and a **Go learning exercise**.
Code should be well-documented with tests.

## Conventions

- **Document all exported types and functions** with Go doc comments.
- **Explain Go idioms and patterns inline** — add comments that teach, not just describe.
- **Unit tests for every public function** — use table-driven tests (idiomatic Go).
- Flat `main` package — extract packages only if the tool grows.

## Build & test

```bash
go build
go test ./...
```

## Configuration

Config lives at `~/.config/pocket-casts-history/config.yaml`.
Output defaults to `~/.local/share/pocket-casts-history/`.

## Credentials

Stored in macOS Keychain under a configurable service name (default: `pocket-casts-api-login`).
The tool never prompts for or stores credentials itself.

```bash
# Store credentials (user does this once):
security add-generic-password -a "user@example.com" -s "pocket-casts-api-login" -w
```

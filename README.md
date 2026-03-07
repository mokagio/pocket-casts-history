# pocket-casts-history

Go CLI tool that extracts your Pocket Casts listening history via the unofficial API and writes it to a local JSON file.

## Build & run

```bash
make run
```

This runs tests, builds, and executes the tool.

On first run, it creates a config file at `~/.config/pocket-casts-history/config.yaml` and prompts for settings.

## Credentials

Credentials are stored in the macOS Keychain under a configurable service name (default: `pocket-casts-api-login`).

For **interactive use** (manual runs):

```bash
security add-generic-password -a "your@email.com" -s "pocket-casts-api-login" -w
```

For **automated use** (launchd, cron), grant `/usr/bin/security` access so it can read the password without a prompt:

```bash
security add-generic-password -a "your@email.com" -s "pocket-casts-api-login" -w "your-password" -T /usr/bin/security
```

## Running as a launchd service

### 1. Create a wrapper script

launchd runs with a minimal `PATH` that doesn't include Homebrew.
A wrapper script ensures the right `PATH` is set.

Example:

```bash
#!/usr/bin/env bash

set -eu

export PATH="/opt/homebrew/bin:$PATH"

cd /path/to/pocket-casts-history

make run
```

Make it executable:

```bash
chmod +x /path/to/wrapper-script.sh
```

### 2. Create the plist

Save to `~/Library/LaunchAgents/<label>.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.example.pocket-casts-history</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/wrapper-script.sh</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>23</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>/path/to/pocket-casts-history.log</string>
    <key>StandardErrorPath</key>
    <string>/path/to/pocket-casts-history.err</string>
</dict>
</plist>
```

### 3. Load the service

```bash
launchctl load ~/Library/LaunchAgents/<label>.plist
```

### 4. Test it

```bash
launchctl kickstart gui/$(id -u)/<label>
```

Check the result:

```bash
launchctl list <label>
```

A `LastExitStatus` of `0` means success.

### Troubleshooting

- **`go: No such file or directory`** — the wrapper script is missing the `PATH` export for Homebrew.
- **Keychain `exit status 36`** — the Keychain item wasn't created with `-T /usr/bin/security`. Delete and re-add it with the `-T` flag.
- **`kickstart` does nothing over SSH** — the service runs in the `Aqua` (GUI) session. Test locally or via Screen Sharing.
- **Logs are empty** — check that `StandardOutPath`/`StandardErrorPath` in the plist point to writable paths.

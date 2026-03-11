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

## Running as a nightly cron job (recommended)

This is the recommended approach on macOS.
launchd skips jobs if the machine is asleep; cron paired with a scheduled wake is more reliable.

### 1. Schedule a daily wake

Tell macOS to wake the machine 5 minutes before the job runs:

```bash
sudo pmset repeat wake MTWRFSU 22:55:00
```

Verify:

```bash
pmset -g sched
```

### 2. Add the cron job

```bash
crontab -e
```

Add:

```
0 23 * * * /path/to/automations/nightly-podcast-history.sh >> /path/to/automations/nightly-podcast-history.log 2>> /path/to/automations/nightly-podcast-history.err
```

### 3. Grant cron Full Disk Access

On macOS, cron (`/usr/sbin/cron`) needs Full Disk Access to run scripts reliably.
Go to **System Settings → Privacy & Security → Full Disk Access** and add `/usr/sbin/cron`.

### Troubleshooting

- **Job never runs** — check Full Disk Access for `/usr/sbin/cron`.
- **`go: No such file or directory`** — the wrapper script must export `PATH` to include Homebrew (`/opt/homebrew/bin`).
- **Keychain `exit status 36`** — see the Credentials section above.
- **`Permission denied`** — the wrapper script lost its execute bit. Run `chmod +x` on it again. This can happen silently if the script is edited by a tool that doesn't preserve permissions.
- **Machine was asleep** — confirm `pmset -g sched` shows the wake schedule; re-run `sudo pmset repeat wake` if missing.

---

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
- **`Permission denied`** — the wrapper script lost its execute bit. Run `chmod +x` on it again. This can happen silently if the script is edited by a tool that doesn't preserve permissions.
- **Logs are empty** — check that `StandardOutPath`/`StandardErrorPath` in the plist point to writable paths.

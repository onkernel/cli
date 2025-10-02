<p align="center">
  <img src="https://raw.githubusercontent.com/onkernel/kernel-images/main/static/images/Kernel-Wordmark_Accent.svg" alt="Kernel Logo" width="55%">
</p>

<p align="center">
  <img alt="GitHub License" src="https://img.shields.io/github/license/onkernel/cli">
  <a href="https://discord.gg/FBrveQRcud"><img src="https://img.shields.io/discord/1342243238748225556?logo=discord&logoColor=white&color=7289DA" alt="Discord"></a>
  <a href="https://x.com/juecd__"><img src="https://img.shields.io/twitter/follow/juecd__" alt="Follow @juecd__"></a>
  <a href="https://x.com/rfgarcia"><img src="https://img.shields.io/twitter/follow/rfgarcia" alt="Follow @rfgarcia"></a>
</p>

# Kernel CLI

The Kernel CLI is a fast, friendly command‑line interface for Kernel — the platform that provides sandboxed, ready‑to‑use Chrome browsers for browser automations and web agents.

Sign up at [onkernel.com](https://www.onkernel.com/) and read the [docs](https://onkernel.com/docs/introduction).

## What's Kernel?

Kernel provides sandboxed, ready-to-use Chrome browsers for browser automations and web agents. This CLI helps you deploy apps, run actions, manage browsers, and access live views.

### What you can do with the CLI

- Deploy and version apps to Kernel
- Invoke app actions (sync or async) and stream logs
- Create, list, view, and delete managed browser sessions
- Get a live view URL for visual monitoring and remote control

## Installation

Install the Kernel CLI using your favorite package manager:

```bash
# Using brew (recommended)
brew install onkernel/tap/kernel

# Using pnpm
pnpm install -g @onkernel/cli

# Using npm
npm install -g @onkernel/cli
```

Verify the installation:

```bash
which kernel
kernel --version
```

## Quick Start

1. **Authenticate with Kernel:**

   ```bash
   kernel login
   ```

2. **Deploy your first app:**

   ```bash
   kernel deploy index.ts
   ```

3. **Invoke your app:**
   ```bash
   kernel invoke my-app action-name --payload '{"key": "value"}'
   ```

## Authentication

### OAuth 2.0 (Recommended)

The easiest way to authenticate is using OAuth:

```bash
kernel login
```

This opens your browser to complete the authentication flow. Your credentials are securely stored and automatically refreshed.

### API Key

You can also authenticate using an API key:

```bash
export KERNEL_API_KEY=<YOUR_API_KEY>
```

Create an API key from the [Kernel dashboard](https://dashboard.onkernel.com).

## Commands Reference

### Global Flags

- `--version`, `-v` - Print the CLI version
- `--no-color` - Disable color output
- `--log-level <level>` - Set log level (trace, debug, info, warn, error, fatal, print)

### Authentication

- `kernel login [--force]` - Login via OAuth 2.0
- `kernel logout` - Clear stored credentials
- `kernel auth` - Check authentication status

### App Deployment

- `kernel deploy <file>` - Deploy an app to Kernel

  - `--version <version>` - Specify app version (default: latest)
  - `--force` - Allow overwriting existing version
  - `--env <KEY=VALUE>`, `-e` - Set environment variables (can be used multiple times)
  - `--env-file <file>` - Load environment variables from file (can be used multiple times)

- `kernel deploy logs <deployment_id>` - Stream logs for a deployment

  - `--follow`, `-f` - Follow logs in real-time (stream continuously)
  - `--since`, `-s` - How far back to retrieve logs. Duration formats: ns, us, ms, s, m, h (e.g., 5m, 2h, 1h30m). Timestamps also supported: 2006-01-02, 2006-01-02T15:04, 2006-01-02T15:04:05, 2006-01-02T15:04:05.000
  - `--with-timestamps`, `-t` - Include timestamps in each log line

- `kernel deploy history [app_name]` - Show deployment history
  - `--limit <n>` - Max deployments to return (default: 100; 0 = all)

### App Management

- `kernel invoke <app> <action>` - Run an app action

  - `--version <version>`, `-v` - Specify app version (default: latest)
  - `--payload <json>`, `-p` - JSON payload for the action
  - `--sync`, `-s` - Invoke synchronously (timeout after 60s)

- `kernel app list` - List deployed apps

  - `--name <app_name>` - Filter by app name
  - `--version <version>` - Filter by version

- `kernel app history <app_name>` - Show deployment history for an app
  - `--limit <n>` - Max deployments to return (default: 100; 0 = all)

### Logs

- `kernel logs <app_name>` - View app logs
  - `--version <version>` - Specify app version (default: latest)
  - `--follow`, `-f` - Follow logs in real-time
  - `--since <time>`, `-s` - How far back to retrieve logs (e.g., 5m, 1h)
  - `--with-timestamps` - Include timestamps in log output

### Browser Management

- `kernel browsers list` - List running browsers
- `kernel browsers create` - Create a new browser session
  - `-p, --persistence-id <id>` - Unique identifier for browser session persistence
  - `-s, --stealth` - Launch browser in stealth mode to avoid detection
  - `-H, --headless` - Launch browser without GUI access
- `kernel browsers delete <id or persistent id>` - Delete a browser
  - `-y, --yes` - Skip confirmation prompt
- `kernel browsers view <id or persistent id>` - Get live view URL for a browser

### Browser Logs

- `kernel browsers logs stream <id or persistent id>` - Stream browser logs
  - `--source <source>` - Log source: "path" or "supervisor" (required)
  - `--follow` - Follow the log stream (default: true)
  - `--path <path>` - File path when source=path
  - `--supervisor-process <name>` - Supervisor process name when source=supervisor. Most useful value is "chromium"

### Browser Replays

- `kernel browsers replays list <id or persistent id>` - List replays for a browser
- `kernel browsers replays start <id or persistent id>` - Start a replay recording
  - `--framerate <fps>` - Recording framerate (fps)
  - `--max-duration <seconds>` - Maximum duration in seconds
- `kernel browsers replays stop <id or persistent id> <replay-id>` - Stop a replay recording
- `kernel browsers replays download <id or persistent id> <replay-id>` - Download a replay video
  - `-o, --output <path>` - Output file path for the replay video

### Browser Process Control

- `kernel browsers process exec <id or persistent id> [--] [command...]` - Execute a command synchronously
  - `--command <cmd>` - Command to execute (optional; if omitted, trailing args are executed via /bin/bash -c)
  - `--args <args>` - Command arguments
  - `--cwd <path>` - Working directory
  - `--timeout <seconds>` - Timeout in seconds
  - `--as-user <user>` - Run as user
  - `--as-root` - Run as root
- `kernel browsers process spawn <id or persistent id> [--] [command...]` - Execute a command asynchronously
  - `--command <cmd>` - Command to execute (optional; if omitted, trailing args are executed via /bin/bash -c)
  - `--args <args>` - Command arguments
  - `--cwd <path>` - Working directory
  - `--timeout <seconds>` - Timeout in seconds
  - `--as-user <user>` - Run as user
  - `--as-root` - Run as root
- `kernel browsers process kill <id or persistent id> <process-id>` - Send a signal to a process
  - `--signal <signal>` - Signal to send: TERM, KILL, INT, HUP (default: TERM)
- `kernel browsers process status <id or persistent id> <process-id>` - Get process status
- `kernel browsers process stdin <id or persistent id> <process-id>` - Write to process stdin (base64)
  - `--data-b64 <data>` - Base64-encoded data to write to stdin (required)
- `kernel browsers process stdout-stream <id or persistent id> <process-id>` - Stream process stdout/stderr

### Browser Filesystem

- `kernel browsers fs new-directory <id or persistent id>` - Create a new directory
  - `--path <path>` - Absolute directory path to create (required)
  - `--mode <mode>` - Directory mode (octal string)
- `kernel browsers fs delete-directory <id or persistent id>` - Delete a directory
  - `--path <path>` - Absolute directory path to delete (required)
- `kernel browsers fs delete-file <id or persistent id>` - Delete a file
  - `--path <path>` - Absolute file path to delete (required)
- `kernel browsers fs download-dir-zip <id or persistent id>` - Download a directory as zip
  - `--path <path>` - Absolute directory path to download (required)
  - `-o, --output <path>` - Output zip file path
- `kernel browsers fs file-info <id or persistent id>` - Get file or directory info
  - `--path <path>` - Absolute file or directory path (required)
- `kernel browsers fs list-files <id or persistent id>` - List files in a directory
  - `--path <path>` - Absolute directory path (required)
- `kernel browsers fs move <id or persistent id>` - Move or rename a file or directory
  - `--src <path>` - Absolute source path (required)
  - `--dest <path>` - Absolute destination path (required)
- `kernel browsers fs read-file <id or persistent id>` - Read a file
  - `--path <path>` - Absolute file path (required)
  - `-o, --output <path>` - Output file path (optional)
- `kernel browsers fs set-permissions <id or persistent id>` - Set file permissions or ownership
  - `--path <path>` - Absolute path (required)
  - `--mode <mode>` - File mode bits (octal string) (required)
  - `--owner <user>` - New owner username or UID
  - `--group <group>` - New group name or GID
- `kernel browsers fs upload <id or persistent id>` - Upload one or more files
  - `--file <local:remote>` - Mapping local:remote (repeatable)
  - `--dest-dir <path>` - Destination directory for uploads
  - `--paths <paths>` - Local file paths to upload
- `kernel browsers fs upload-zip <id or persistent id>` - Upload a zip and extract it
  - `--zip <path>` - Local zip file path (required)
  - `--dest-dir <path>` - Destination directory to extract to (required)
- `kernel browsers fs write-file <id or persistent id>` - Write a file from local data
  - `--path <path>` - Destination absolute file path (required)
  - `--mode <mode>` - File mode (octal string)
  - `--source <path>` - Local source file path (required)

### Browser Extensions

- `kernel browsers extensions upload <id or persistent id> <extension-path>...` - Ad-hoc upload of one or more unpacked extensions to a running browser instance.

### Extension Management

- `kernel extensions list` - List all uploaded extensions
- `kernel extensions upload <directory>` - Upload an unpacked browser extension directory
  - `--name <name>` - Optional unique extension name
- `kernel extensions download <id-or-name>` - Download an extension archive
  - `--to <directory>` - Output directory (required)
- `kernel extensions download-web-store <url>` - Download an extension from the Chrome Web Store
  - `--to <directory>` - Output directory (required)
  - `--os <os>` - Target OS: mac, win, or linux (default: linux)
- `kernel extensions delete <id-or-name>` - Delete an extension by ID or name
  - `-y, --yes` - Skip confirmation prompt

## Examples

### Deploy with environment variables

```bash
# Set individual variables
kernel deploy index.ts --env API_KEY=abc123 --env DEBUG=true

# Load from .env file
kernel deploy index.ts --env-file .env

# Combine both methods
kernel deploy index.ts --env-file .env --env OVERRIDE_VAR=value
```

### Invoke with payload

```bash
# Simple invoke
kernel invoke my-scraper scrape-page

# With JSON payload
kernel invoke my-scraper scrape-page --payload '{"url": "https://example.com"}'

# Synchronous invoke (wait for completion)
kernel invoke my-scraper quick-task --sync
```

### Follow logs in real-time

```bash
# Follow logs
kernel logs my-app --follow

# Show recent logs with timestamps
kernel logs my-app --since 1h --with-timestamps
```

### Browser management

```bash
# List all browsers
kernel browsers list

# Create a new browser session
kernel browsers create

# Create a persistent browser session
kernel browsers create --persistence-id my-browser-session

# Create a headless browser in stealth mode
kernel browsers create --headless --stealth

# Delete a persistent browser
kernel browsers delete --by-persistent-id my-browser-session --yes

# Get live view URL
kernel browsers view --by-id browser123

# Stream browser logs
kernel browsers logs stream my-browser --source supervisor --follow --supervisor-process chromium

# Start a replay recording
kernel browsers replays start my-browser --framerate 30 --max-duration 300

# Execute a command in the browser VM
kernel browsers process exec my-browser -- ls -alh /tmp

# Upload files to the browser VM
kernel browsers fs upload my-browser --file "local.txt:remote.txt" --dest-dir "/tmp"

# List files in a directory
kernel browsers fs list-files my-browser --path "/tmp"
```

### Extension management

```bash
# List all uploaded extensions
kernel extensions list

# Upload an unpacked extension directory
kernel extensions upload ./my-extension --name my-custom-extension

# Download an extension from Chrome Web Store
kernel extensions download-web-store "https://chrome.google.com/webstore/detail/extension-id" --to ./downloaded-extension

# Download a previously uploaded extension
kernel extensions download my-extension-id --to ./my-extension

# Delete an extension
kernel extensions delete my-extension-name --yes

# Upload extensions to a running browser instance
kernel browsers extensions upload my-browser ./extension1 ./extension2
```

## Getting Help

- `kernel --help` - Show all available commands
- `kernel <command> --help` - Get help for a specific command

## Documentation

For complete documentation, visit:

- [📖 Documentation](https://onkernel.com/docs)
- [🚀 Quickstart Guide](https://onkernel.com/docs/quickstart)
- [📋 CLI Reference](https://onkernel.com/docs/reference/cli)

## Support

- [Discord Community](https://discord.gg/kernel)
- [GitHub Issues](https://github.com/onkernel/kernel/issues)
- [Documentation](https://onkernel.com/docs)

---

For development and contribution information, see [DEVELOPMENT.md](./DEVELOPMENT.md).

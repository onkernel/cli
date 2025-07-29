# Kernel CLI

The Kernel CLI helps you deploy and run web automation apps on the Kernel platform. Build browser automation, web scraping, and AI agents that run in the cloud.

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

### App Management

- `kernel deploy <file>` - Deploy an app to Kernel
  - `--version <version>` - Specify app version (default: latest)
  - `--force` - Allow overwriting existing version
  - `--env <KEY=VALUE>`, `-e` - Set environment variables (can be used multiple times)
  - `--env-file <file>` - Load environment variables from file (can be used multiple times)

- `kernel invoke <app> <action>` - Run an app action
  - `--version <version>`, `-v` - Specify app version (default: latest)
  - `--payload <json>`, `-p` - JSON payload for the action
  - `--sync`, `-s` - Invoke synchronously (timeout after 60s)

- `kernel app list` - List deployed apps
  - `--name <app_name>` - Filter by app name
  - `--version <version>` - Filter by version

- `kernel app history <app_name>` - Show deployment history for an app

### Logs

- `kernel logs <app_name>` - View app logs
  - `--version <version>` - Specify app version (default: latest)
  - `--follow`, `-f` - Follow logs in real-time
  - `--since <time>`, `-s` - How far back to retrieve logs (e.g., 5m, 1h)
  - `--with-timestamps` - Include timestamps in log output

### Browser Management

- `kernel browsers list` - List running browsers
- `kernel browsers create` - Create a new browser session
  - `--persistence-id <id>` - Unique identifier for browser session persistence
  - `--stealth` - Launch browser in stealth mode to avoid detection
  - `--headless` - Launch browser without GUI access
- `kernel browsers delete` - Delete a browser
  - `--by-persistent-id <id>` - Delete by persistent ID
  - `--by-id <id>` - Delete by session ID
  - `--yes`, `-y` - Skip confirmation prompt
- `kernel browsers view` - Get live view URL for a browser
  - `--by-persistent-id <id>` - View by persistent ID
  - `--by-id <id>` - View by session ID

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
```

## Getting Help

- `kernel --help` - Show all available commands
- `kernel <command> --help` - Get help for a specific command

## Documentation

For complete documentation, visit:

- [📖 Documentation](https://docs.onkernel.com)
- [🚀 Quickstart Guide](https://docs.onkernel.com/quickstart)
- [📋 CLI Reference](https://docs.onkernel.com/reference/cli)

## Support

- [Discord Community](https://discord.gg/kernel)
- [GitHub Issues](https://github.com/onkernel/kernel/issues)
- [Documentation](https://docs.onkernel.com)

---

For development and contribution information, see [DEVELOPMENT.md](./DEVELOPMENT.md).

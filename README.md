# GitHub–Todoist Sync

GitHub–Todoist Sync is a tool for automatic bidirectional synchronization between GitHub Issues and tasks in Todoist. Each issue is mapped to a task in Todoist, including priority (derived from labels), state synchronization (open/closed ↔ incomplete/complete), and a backlink to the original issue. It supports one-time runs, one-way modes, and a continuous daemon mode. On first sync, it can automatically create a project in Todoist if it doesn't exist. Configuration is done via a simple `.env` file, and convenient commands are available in the Makefile for both development and production.

## Features

- Bidirectional sync: GitHub ↔ Todoist
- Issue → Task mapping: each GitHub issue becomes a Todoist task
- Priority from labels: urgent → 4, high → 3, medium → 2, low → 1 (default 1)
- State sync: closing issue completes task and vice versa
- Automatic Todoist project creation on first sync
- One-shot, one-way and daemon modes
- Simple configuration via environment variables and clear logs

## Requirements

- Go 1.24+
- GitHub Personal Access Token (scopes: `repo`, `read:project`)
- Todoist API Token

## Quick Start

### Prepare environment

```bash
make dev
```

### Edit environment with tokens

```bash
nano .env
```

### One-shot sync

```bash
make run
```

## Configuration (`.env` variables)

```bash
# GitHub
GITHUB_TOKEN=ghp_your_github_token_here
GITHUB_OWNER=your_username
GITHUB_REPO=your_repository_name

# Todoist
TODOIST_TOKEN=your_todoist_api_token_here
TODOIST_PROJECT_NAME=GitHub Sync

# App
SYNC_INTERVAL_MINUTES=15
DEBUG=true
```

- Obtain a GitHub Personal Access Token with `repo` and `read:project` scopes.
- Get a Todoist API token in Todoist → Settings → Integrations → Developer.
- `SYNC_INTERVAL_MINUTES` sets the interval for daemon mode; `DEBUG` enables detailed logging.

## Usage (Makefile commands)

```bash
make help
make setup
make deps
make build
make run
make github-sync
make todoist-sync
make daemon
make stop
make logs
make status
make clean
```

## Direct Run

### Build

```bash
go build -o bin/github-todoist-sync cmd/sync/main.go
```

### One-shot sync

```bash
./bin/github-todoist-sync -mode=once -verbose
```

### Daemon (periodic sync)

```bash
./bin/github-todoist-sync -mode=daemon -verbose
```

### GitHub → Todoist only

```bash
./bin/github-todoist-sync -mode=github-only -verbose
```

### Todoist → GitHub only

```bash
./bin/github-todoist-sync -mode=todoist-only -verbose
```

## How Sync Works

- **GitHub → Todoist:**
    - New issues create corresponding Todoist tasks
    - Title: issue title → task content
    - Description: contains link to GitHub issue
    - Priority from labels: urgent → 4, high → 3, medium → 2, low → 1 (default 1)
    - Labels: GitHub labels → Todoist labels
    - State: closed issue → completed task

- **Todoist → GitHub:**
    - Completed task → closes corresponding GitHub issue
    - Reopened task → reopens GitHub issue

## Logging

```bash
DEBUG=true
make logs
tail -f logs/sync.log
```

## Troubleshooting

- **"GITHUB_TOKEN is required"**: check `.env` and token scopes (`repo`, `read:project`)
- **"project not found"**: app can create project in Todoist; check `TODOIST_PROJECT_NAME`
- **"API error 403"**: token expired or insufficient permissions; generate new token
- **Verbose run for diagnostics:**
  ```bash
  ./bin/github-todoist-sync -mode=once -verbose
  ```
  or:
  ```bash
  DEBUG=true make run
  ```

## Project Structure

```
github-todoist-sync/
├── cmd/sync/main.go
├── internal/
│   ├── config/config.go
│   ├── github/client.go
│   ├── todoist/client.go
│   └── sync/service.go
├── .env.example
├── Makefile
├── go.mod
└── README.md
```

## Examples

```bash
make run
make daemon
make logs
make stop
make github-sync
```

## Security

- Keep API tokens safe and never commit `.env`
- Use minimal required scopes
- Rotate tokens regularly

## License

MIT License — see LICENSE

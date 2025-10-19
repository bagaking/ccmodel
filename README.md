# ccmodel

ccmodel is a command-line manager for switching and inspecting AI model configurations, built with [cmdux](https://github.com/bagaking/cmdux).

## Features

- **Model Management**: List, switch, and manage AI models
- **Rich UI**: Beautiful boxes, tables, and interactive elements
- **Real-time Monitoring**: Interactive `top` mode for live quota monitoring
- **JSON API**: Machine-readable output for integration with editors and tools
- **Quota Monitoring**: Real-time API usage tracking with configurable macros
- **Shell Integration**: Comprehensive shell completion support

## Installation

```bash
go install github.com/bagaking/ccmodel@latest
```

Or build from source:

```bash
git clone https://github.com/bagaking/ccmodel
cd ccmodel
go build -o ccmodel
```

## Completion

```zsh
eval "$(ccmodel completion zsh 2>/dev/null || true)"
```

## Usage

### Basic Commands

```bash
# Show welcome screen with quick start guide
ccmodel

# List all available models
ccmodel list

# Show current active model (human-readable)
ccmodel current

# Show current active model (JSON format)
ccmodel current --json

# Switch to a different model
ccmodel <model_name>

# Launch Claude CLI with session tracking (pass flags after --)
ccmodel exec claude -- --help

# Interactive real-time monitoring
ccmodel top
```

### Quota Monitoring

When models include quota configuration macros, real-time usage information is displayed:

```bash
# View detailed quota for current model
ccmodel current
# ╭[ 📊 API Quota Status ]╮
# │ Total Quota: 1000     │
# │ Used: 250 (25.0%)     │
# │ Remaining: 750        │
# ╰───────────────────────╯

# Quick quota overview for all models
ccmodel list
# Shows quota in table: "250/1000 (25%)"

# Real-time interactive monitoring
ccmodel top
# Interactive dashboard with live updates
```

### JSON API for Integrations

For VS Code extensions, status bar widgets, and other integrations:

```bash
# Get current model status as JSON
ccmodel current --json
ccmodel status --json  # Equivalent alias

# Example output:
{
  "model": "claude-sonnet",
  "config_path": "~/.claude/settings.json",
  "file_size": 286,
  "last_modified": "2025-09-07T23:58:25Z",
  "is_active": true,
  "is_custom": false,
  "quota": {
    "Total": 1000.0,
    "Used": 250.5
  }
}
```

### Real-time Monitoring (Top Mode)

Interactive dashboard for monitoring all models simultaneously:

```bash
ccmodel top [--interval 15s] [--colors auto|light|dark]

# Features:
# • Live quota updates for all models
# • Interactive model switching (1-9 keys)
# • Adaptive color schemes (auto-detects terminal background)
# • Process collaboration (shares data between instances)
# • Configurable refresh intervals (minimum 10s)
```

### Command Reference

| Command | Aliases | Description | JSON Support |
|---------|---------|-------------|--------------|
| `ccmodel` | - | Show welcome screen and quick start guide | ❌ |
| `ccmodel list` | - | List all available model configurations | ❌ |
| `ccmodel current` | `status`, `whoami` | Show current active model status | ✅ `--json` |
| `ccmodel switch <model>` | `<model>` | Switch to specified model | ❌ |
| `ccmodel exec <target>` | - | Proxy launch for Claude/Codex CLIs with session recording | ❌ |
| `ccmodel top` | `monitor`, `watch` | Interactive real-time monitoring dashboard | ❌ |
| `ccmodel backup` | - | Backup current configuration | ❌ |
| `ccmodel completion` | - | Generate shell completion scripts | ❌ |

### Proxy Execution Sessions

Use `ccmodel exec` to launch the native Claude or Codex CLI while ccmodel records the run:

```bash
# Launch Claude and forward flags after --
ccmodel exec claude -- --workspace my-project

# Launch Codex (Cursor) CLI with custom path override
CCMODEL_EXEC_CODEX=/Applications/Cursor.app/Contents/MacOS/codex \
  ccmodel exec codex -- --reset
```

- All proxy sessions are stored under `~/.claude/ccmodel/exec_sessions/`.
- Metadata includes command line, working directory, active model, and exit status to help restore runs.
- Override the binary with `CCMODEL_EXEC_CLAUDE` or `CCMODEL_EXEC_CODEX` when the executable is outside `$PATH`.

### Screenshots

```
╭───────────────────────[ AI MODEL REGISTRY ]────────────────────────╮
│ Available configurations for Claude Code                           │
╰────────────────────────────────────────────────────────────────────╯
●  Status: k2

╭───┬────────┬────────────┬──────┬──────────────┬─────────────────┬────────╮
│ # │ Status │ Model Name │ Size │ Modified     │ Quota           │ State  │
├───┼────────┼────────────┼──────┼──────────────┼─────────────────┼────────┤
│ 1 │ ★      │ k2         │ 296B │ Jul 19 01:41 │ 150/500 (30%)   │ ACTIVE │
│ 2 │ ○      │ claude     │ 286B │ Jul 19 01:56 │ -               │        │
╰───┴────────┴────────────┴──────┴──────────────┴─────────────────┴────────╯
📁  Config Path: ~/.claude
📊  Total Models: 2
```
 
## Development

### Project Structure

```
ccmodel/
├── cmd/                # commands
├───── ...
├── main.go             # Main application using cmdux
├── go.mod              # Dependencies (includes cmdux)
├── ...
└── README.md           # This file
```


### Development Commands

The project includes comprehensive development tooling via Make:

```bash
# Development
make dev          # Run in development mode
make test         # Run all tests  
make fmt          # Format Go code
make lint         # Run golangci-lint

# Building
make build        # Build binary for current platform
make build-all    # Cross-compile for all platforms
make clean        # Clean build artifacts

# Installation  
make install      # Install to /usr/local/bin (requires sudo)
make install-dev  # Install to ~/go/bin for development
make uninstall    # Remove from /usr/local/bin

# Release
make release      # Prepare release with all binaries
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Configuration Macros

ccmodel supports configuration macros using the `__cc` field, which acts as a "macro" system for extending functionality without affecting core configuration management.

### Quota Monitoring Macro

Add quota monitoring to any model configuration by including a `__cc` section:

```json
{
  "apiKey": "your-api-key",
  "model": "claude-3-5-sonnet-20241022",
  "env": {
    "ANTHROPIC_API_KEY": "your-actual-key",
    "CUSTOM_TOKEN": "your-token"
  },
  "__cc": {
    "quota_test": {
      "post": {
        "url": "https://api.example.com/quota/check",
        "timeout": 3, // 8s by default
        "header": {
          "accept": "*/*",
          "content-type": "application/json",
          "authorization": "Bearer $token"
        },
        "query": {
          "$token_key": "model",
          "page": "1",
          "page_size": "10"
        },
        "data": {
          "$key": "env.ANTHROPIC_API_KEY",
          "$token": "env.CUSTOM_TOKEN",
          "timestamp": "2025-01-01"
        },
        "result": {
          "total": "data.quotaLimit",
          "used": "data.quotaUsed"
        }
      }
    }
  }
}
```

### Macro Features

- **Variable Expansion**: Use `$key` syntax to reference values from anywhere in the configuration
  - `"$key": "env.ANTHROPIC_API_KEY"` → Looks up `env.ANTHROPIC_API_KEY` from config
  - `"$token": "auth.bearer_token"` → Looks up `auth.bearer_token` from config
- **JSON Path Results**: Use gjson path syntax to extract quota information
  - `"total": "data.quotaLimit"` → Extracts from `response.data.quotaLimit`
  - `"used": "usage.current"` → Extracts from `response.usage.current`
- **Automatic Display**: Quota info appears in both `ccmodel current` and `ccmodel list`
- **Performance Optimized**: Concurrent requests with configurable timeouts

### Macro Isolation

- **Copy Protection**: `__cc` fields are automatically filtered when switching configurations
- **Comparison Ignored**: File checksums ignore macro fields for clean model matching
- **Clean Storage**: Core configurations remain uncluttered while macros provide extended functionality

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework and command structure
- [gjson](https://github.com/tidwall/gjson) - JSON path queries for macro expansion  
- [cmdux](https://github.com/bagaking/cmdux) - The terminal UI library powering this application
    - ✨ Beautiful terminal UI with rich animations
    - 🎨 Adaptive color schemes with auto-detection
    - 📊 Enhanced tables and data visualization
    - 🚀 Smooth loading animations and progress bars
    - 🎯 Better user experience and interaction

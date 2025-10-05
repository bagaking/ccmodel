# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ccmodel is a CLI tool for managing AI model configurations for Claude Code. It enables users to switch between different AI service providers (OpenRouter, Moonshot, Anthropic, etc.) by swapping settings.json files atomically.

Built with:
- **Language**: Go 1.23+
- **CLI Framework**: Cobra
- **UI Library**: cmdux (terminal UI with rich animations and themes)
- **Dependencies**: fatih/color for terminal colors

## Development Commands

### Core Development
- `make build` - Build binary for current platform (output: `build/ccmodel`)
- `make test` - Run all tests
- `make fmt` - Format Go code
- `make lint` - Run golangci-lint (installs if missing)
- `make clean` - Clean build artifacts

### Development Workflow
- `make dev` - Run in development mode (runs `go run . list`)
- `make quick-test` - Quick test with sample configs
- `make install-dev` - Install development binary to `~/go/bin/`

### Installation
- `make install` - Install to `/usr/local/bin/ccmodel` (requires sudo)
- `make uninstall` - Remove from `/usr/local/bin/`
- `go install github.com/bagaking/ccmodel@latest` - Install latest release

### Release Management
- `make build-all` - Cross-compile for all platforms (macOS, Linux, Windows)
- `make release` - Prepare release (builds all platforms, suggests gh command)

## Project Architecture

### Command Structure
- `cmd/root.go` - Main command and model switching logic
- `cmd/list.go` - List available models with rich UI
- `cmd/current.go` - Show current active model
- `cmd/switch.go` - Switch between models
- `cmd/backup.go` - Backup/restore functionality
- `cmd/completion.go` - Shell completion support

### Configuration Management
- Configuration directory: `~/.claude/`
- Model configs: `settings.<model>.json` files
- Active config: `settings.json` (symlink to active model)

### Key Features
- **Atomic Switching**: Safe model switching via symlinks
- **Rich Terminal UI**: Powered by cmdux with animations and themes
- **Shell Completion**: Support for bash/zsh/fish completion
- **Cross-platform**: Builds for macOS, Linux, Windows (ARM64 & AMD64)
- **Quota Monitoring**: Real-time API quota checking with HTTP requests

## Testing

Run tests with `make test` or `go test ./...`. The project includes:
- Unit tests for core functionality
- Integration tests for configuration management
- Quick test suite with mock configurations

## Build System

The Makefile provides comprehensive build automation:
- Development builds with version info from git
- Cross-compilation for multiple platforms
- Automated dependency management
- Release preparation workflows

## Code Style

- Follow standard Go conventions
- Use `make fmt` for consistent formatting
- Run `make lint` before committing
- All commands use cobra framework patterns
- Terminal UI components use cmdux library

## Quota Monitoring Feature

The quota monitoring feature allows real-time API usage tracking by configuring HTTP requests in model configurations. See the [Configuration Macros](README.md#configuration-macros) section in README.md for complete documentation.

### Configuration Format

Add `__ccmodel` or `__cc` (shorthand) section to any model configuration:

```json
{
  "apiKey": "your-api-key",
  "model": "claude-3-5-sonnet-20241022",
  "__cc": {
    "quota_test": {
      "post": {
        "url": "https://api.example.com/quota",
        "header": {
          "content-type": "application/json",
          "authorization": "Bearer $env.API_TOKEN"
        },
        "data": {
          "key": "$env.ANTHROPIC_API_KEY"
        },
        "result": {
          "total": "data.totalQuota",
          "used": "data.usedQuota"
        }
      }
    }
  }
}
```

### Features

- **Environment Variables**: Use `$env.VAR_NAME` syntax for secure credential handling
- **JSON Path Extraction**: Uses gjson library for flexible response parsing
- **Visual Feedback**: Color-coded quota display (green < 70%, yellow < 90%, red ≥ 90%)
- **Error Handling**: Graceful degradation when quota API is unavailable
- **Automatic Display**: Shows in both `ccmodel current` and `ccmodel list` commands
- **Performance Optimized**: Concurrent quota fetching with short timeout (3s) for list view

### JSON Path Examples

- Simple paths: `"total"`, `"used"`  
- Nested paths: `"data.quota.total"`, `"response.usage.consumed"`
- Array access: `"quotas.0.limit"`, `"data.metrics.1.value"`

### Usage Examples

#### Current Model Status (Detailed View)
```bash
ccmodel current
```
Shows detailed quota information with color-coded status box.

#### List All Models with Quota Summary
```bash
ccmodel list
```
Shows quota in compact table format: `25.0K/100.0K (25%)`
- Uses 3-second timeout for quick response
- Fetches quotas concurrently for all models
- Models without quota config show `-`
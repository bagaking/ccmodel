# ccmodel

A powerful AI Model Configuration Manager built with [cmdux](https://github.com/bagaking/cmdux).

## Features

- **Model Management**: List, switch, and manage AI models
- **Rich UI**: Beautiful boxes, tables, and interactive elements
- **Animations**: Loading spinners, progress bars, and visual effects
- **Themes**: Multiple built-in themes for different preferences
- **Demo Mode**: Showcase advanced terminal effects
- **Quota Monitoring**: Real-time API usage tracking with configurable macros

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

# Show current active model
ccmodel current

# Switch to a different model
ccmodel <model_name>
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
```

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
📁  Config Path: /Users/bytedance/.claude
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


### Building

```bash
go build -o ccmodel
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

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

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [color](https://github.com/fatih/color) - Terminal colors
- [gjson](https://github.com/tidwall/gjson) - JSON path queries for macro expansion
- [cmdux](https://github.com/bagaking/cmdux) - The terminal UI library powering this application
    - ✨ Beautiful terminal UI with rich animations
    - 🎨 Multiple theme support (Default, Dark, Cyberpunk, Monochrome)
    - 📊 Enhanced tables and data visualization
    - 🚀 Smooth loading animations and progress bars
    - 🎯 Better user experience and interaction
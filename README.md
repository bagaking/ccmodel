# ccmodel

A powerful AI Model Configuration Manager built with [cmdux](https://github.com/bagaking/cmdux).

## Features

- **Model Management**: List, switch, and manage AI models
- **Rich UI**: Beautiful boxes, tables, and interactive elements
- **Animations**: Loading spinners, progress bars, and visual effects
- **Themes**: Multiple built-in themes for different preferences
- **Demo Mode**: Showcase advanced terminal effects

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

### Screenshots

```
╭───────────────────────[ AI MODEL REGISTRY ]────────────────────────╮
│ Available configurations for Claude Code                           │
╰────────────────────────────────────────────────────────────────────╯
●  Status: k2

╭───┬────────┬────────────┬──────┬──────────────┬────────╮
│ # │ Status │ Model Name │ Size │ Modified     │ State  │
├───┼────────┼────────────┼──────┼──────────────┼────────┤
│ 1 │ ★      │ k2         │ 296B │ Jul 19 01:41 │ ACTIVE │
│ 2 │ ○      │ claude     │ 286B │ Jul 19 01:56 │        │
╰───┴────────┴────────────┴──────┴──────────────┴────────╯
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

## Dependencies


- [cobra](https://github.com/spf13/cobra) - CLI framework
- [color](https://github.com/fatih/color) - Terminal colors
- [cmdux](https://github.com/bagaking/cmdux) - The terminal UI library powering this application
    - ✨ Beautiful terminal UI with rich animations
    - 🎨 Multiple theme support (Default, Dark, Cyberpunk, Monochrome)
    - 📊 Enhanced tables and data visualization
    - 🚀 Smooth loading animations and progress bars
    - 🎯 Better user experience and interaction
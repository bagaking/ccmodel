# ccmodel

A simple CLI tool to manage and switch between different Claude Code model configurations.  
Inspired by tools like `nvm` and `pyenv`.

---

## Features

- Switch between multiple model/service configurations instantly
- Atomic swaps with automatic backups
- Cross-platform: macOS, Linux, Windows
- Shell completion for bash, zsh, fish
- No dependencies, single binary

---

## Quick Start

```bash
# Install (macOS/Linux)
curl -sSL https://raw.githubusercontent.com/bagaking/ccmodel/main/install.sh | bash

# Or download from releases: https://github.com/bagaking/ccmodel/releases

# List available models
ccmodel list

# Switch to a model
ccmodel switch gpt4
ccmodel switch claude3

# Show current model
ccmodel current
```

---

## Installation

### Homebrew (macOS/Linux)
```bash
brew install bagaking/tap/ccmodel
```

### Go Install
```bash
go install github.com/bagaking/ccmodel@latest
```

### Manual
1. Download from [releases](https://github.com/bagaking/ccmodel/releases)
2. Extract and add to PATH
3. Run `ccmodel --help`

---

## Usage

### Basic Commands

```bash
ccmodel list         # List all available models
ccmodel ls           # Alias for list

ccmodel current      # Show current model
ccmodel status       # Alias for current

ccmodel switch NAME  # Switch to a model

ccmodel backup       # Backup current configuration
```

### Shell Completion

```bash
# bash
# 推荐在 ~/.bashrc 末尾添加如下内容，并 source 使其生效

echo 'eval "$(ccmodel completion bash)"' >> ~/.bashrc
source ~/.bashrc

# zsh
# 推荐在 ~/.zshrc 末尾添加如下内容，并 source 使其生效
# 你可以直接复制以下命令到终端：

echo 'eval "$(ccmodel completion zsh)"' >> ~/.zshrc
source ~/.zshrc

# fish
# 推荐在 ~/.config/fish/config.fish 末尾添加如下内容

ccmodel completion fish | source >> ~/.config/fish/config.fish
```

---

## Configuration

### Adding New Models

1. Create a new JSON file in `~/.claude/`
2. Name it `settings.{model-name}.json`
3. Use ccmodel to switch:

```bash
cat << EOF > ~/.claude/settings.openrouter.json
{
  "env": {
    "ANTHROPIC_API_KEY": "your-key-here",
    "ANTHROPIC_BASE_URL": "https://openrouter.ai/api/v1"
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}
EOF

ccmodel switch openrouter
```

---

## Example Output

```
Available Models:
  1. gpt4         1.2KB  2024-01-15 14:32  [active]
  2. claude3      1.1KB  2024-01-14 09:15
  3. openrouter   1.3KB  2024-01-13 18:45

Config directory: /Users/you/.claude
Total models: 3
```

---

## Development

### Build from Source

```bash
git clone https://github.com/bagaking/ccmodel.git
cd ccmodel
go mod download
go build -o ccmodel .
```

### Run Tests

```bash
go test ./...
```

---

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Acknowledgments

- Inspired by `nvm`, `pyenv`, and similar tools
- Built with [Cobra](https://github.com/spf13/cobra)
- Colors via [Fatih/color](https://github.com/fatih/color)

---

Maintained by contributors.
# RepoWalk 🚶‍♂️

[![NPM Version](https://img.shields.io/npm/v/repowalk.svg)](https://www.npmjs.com/package/repowalk)
[![NPM Downloads](https://img.shields.io/npm/dt/repowalk.svg)](https://www.npmjs.com/package/repowalk)
[![Go Version](https://img.shields.io/github/go-mod/go-version/QubeUtils/repowalk)](https://github.com/QubeUtils/repowalk)
[![Built with Charm](https://img.shields.io/badge/Built%20with-Charm-4D69FF?style=flat-square&logo=go)](https://charm.sh/)
[![License](https://img.shields.io/github/license/QubeUtils/repowalk)](https://github.com/QubeUtils/repowalk/blob/main/LICENSE)

RepoWalk is a blazing-fast, interactive CLI tool written in Go that walks your repository and packs it into a beautifully formatted Markdown document. 

Designed specifically for the era of AI coding, RepoWalk is the ultimate bridge between your local codebase and large language models (LLMs) like Claude, ChatGPT, or Ollama. Instead of copy-pasting individual files or struggling with context windows, you can generate a single optimized "Super Context" file containing exactly what the AI needs to know.

![Demo](Demo.gif)

## ✨ Why RepoWalk?

**Copy-paste is broken for large codebases.** RepoWalk solves this by:
- 🎯 Automatically filtering boilerplate and noise
- 🔐 Redacting secrets and API keys before sharing with LLMs
- 📊 Showing exact token counts so you know what fits in your prompt window
- 🔍 Letting you handpick files interactively in a beautiful TUI
- ⚡ Processing monorepos in milliseconds, not minutes
- 📝 Exporting to Markdown, JSON, or YAML—whatever your workflow needs

## 🌟 Features

*   **⚡ Blazing Fast**: Multi-threaded traversal handles massive monorepos in milliseconds.
*   **🖥️ Beautiful Interactive TUI**: Navigate a visual tree of your repository. Use the arrow keys to seamlessly toggle files and directories.
*   **👀 Interactive File Preview & Line Stripping**: View file contents side-by-side in real-time. Enter Edit Mode to strip out boilerplate lines directly from the UI with Tri-State toggling (Include / Strike-through Exclude / Green Exclusive Inclusion).
*   **💾 Flexible Export Options**: Quick Save to your default output, or use the interactive `Save As...` workflow to customize filename and directory on the fly.
*   **🌳 Colored CLI Outputs**: Run `repowalk tree` or `repowalk stats` for beautiful, terminal-native styled outputs using Lipgloss.
*   **🔢 Real-time Token Counting**: Integrates `tiktoken` to give you exact token counts of your payload before you send it to an LLM.
*   **🛡️ Automatic Secret Redaction**: Built-in regex filters catch and mask API keys and secrets so you don't leak them.
*   **🎭 LLM Personas**: Automatically wrap your generated context with engineered prompts for tasks like `security`, `refactor`, or `review`.
*   **📋 Native Clipboard Integration**: Instantly copies the output to your clipboard for quick pasting.
*   **📁 Monorepo-Ready**: Selectively walk specific microservices or subdirectories with `-m` flag.
*   **✅ `.gitignore` Respect**: Only includes files your VCS would track.

## 🚀 Installation

**Homebrew (macOS / Linux):**
```bash
brew install qubeutils/tap/repowalk
```

**NPM (Node.js):**
```bash
npx repowalk ui
# OR install globally
npm install -g repowalk
```

**Go Developer:**
```bash
go install github.com/QubeUtils/repowalk@latest
```

## ⚡ Quick Start

```bash
# Launch the interactive TUI
repowalk ui

# See a tree preview of what will be included
repowalk tree

# Get repo statistics and token count
repowalk stats

# Generate context with a security focus
repowalk --persona security --json > context.json

# Show only recently changed files
repowalk diff > changes.md
```

## 🎮 Usage

### 1. The Interactive TUI (Recommended)

Launch the beautiful terminal UI to visually select exactly what you want to include in your LLM context:

```bash
repowalk ui
```

**Keyboard Shortcuts:**
*   `↑/↓/←/→` or `k/j/h/l`: Navigate the file tree
*   `Space`: Toggle select / deselect
*   `Space` (double-tap): **Exclusive Select** (only this file/folder)
*   `e`: Enter **Edit Mode** to preview and refine line-by-line
*   `Space` (in Edit Mode): Cycle through line states (Include → Exclude → Exclusively Include)
*   `Enter`: **Quick Save** to your default output path
*   `s`: **Save As...** prompt for custom filename and location
*   `c`: Copy generated Markdown to clipboard
*   `/`: Search and filter files (e.g., `.go yaml`)
*   `m`: Toggle **Minimal UI** mode
*   `h` or `?`: Show help screen
*   `PgUp/PgDn`: Scroll file preview
*   `q` or `Esc`: Quit

### 2. The CLI

Run RepoWalk headless to generate context files in your workflows or automation:

**Basic Usage:**
```bash
# Output to stdout
repowalk

# Save to a file
repowalk > repowalk_context.md

# Walk a specific directory
repowalk ./path/to/my/project > repowalk_context.md
```

**Commands:**

| Command | Description |
| :--- | :--- |
| `ui` | Launches an interactive TUI to visually browse and select files, then generate an LLM context dump. |
| `tree` | Prints a colored tree structure of the repository, respecting `.gitignore`. Great for previewing what will be included. |
| `stats` | Shows statistics: total files, total size, and estimated LLM token count. |
| `diff` | Creates a context dump containing only changed files (untracked, unstaged, staged). Perfect for PR reviews. |
| `init` | Creates a `.repowalk.json` config file in your directory for local project-specific settings. |
| `completion` | Generates shell autocompletion scripts. |
| `help` | Shows help for any command. |

**Global Flags:**

| Flag | Description |
| :--- | :--- |
| `--ignore-exts strings` | Override extensions to ignore (comma-separated). Overrides `~/.repowalk/config.json`. |
| `--json` | Export context as JSON format. |
| `--max-size int` | Maximum file size to include in bytes (default 1048576). |
| `-m, --microservice strings`| Walk only specific subdirectories (e.g., `-m auth,payment`). |
| `--no-redact` | Disable automatic secret redaction. |
| `--persona string` | Wrap context with an engineered prompt: `security`, `refactor`, `review`, etc. |
| `--template string` | Provide a custom template for output formatting. |
| `--yaml` | Export context as YAML format. |
| `-h, --help` | Show command help. |

**Tree Command Flags:**

| Flag | Description |
| :--- | :--- |
| `--compact` | Print a compact tree representation. |
| `-L, --level int` | Limit tree depth (e.g., `-L 2` shows 2 levels only). |

### Examples

```bash
# Generate a security-focused context as JSON
repowalk --persona security --json > context.json

# Walk specific microservices, excluding markdown and text
repowalk -m auth,payment --ignore-exts .md,.txt > context.md

# Preview tree up to 2 levels deep
repowalk tree -L 2

# Show stats for the entire repo
repowalk stats

# Generate context for only changed files
repowalk diff > changes-for-review.md

# Copy to clipboard immediately
repowalk ui
# Then press 'c' in the TUI
```

## ⚙️ Configuration

Create a `.repowalk.json` file in your project root to configure RepoWalk locally (this file can be committed to your VCS):

```json
{
  "ignore_exts": [".test.ts", ".spec.js", ".map"],
  "ignore_paths": ["node_modules", ".git", "dist", "build"],
  "max_file_size": 1048576,
  "default_persona": "refactor",
  "output_format": "markdown",
  "redact_secrets": true
}
```

Global config lives in `~/.repowalk/config.json` and is overridden by local `.repowalk.json` settings.

## ❓ FAQ

**Q: Does RepoWalk respect `.gitignore`?**
A: Yes. RepoWalk only includes files that git would track, so your `.gitignore` rules are automatically respected.

**Q: Can it handle monorepos?**
A: Absolutely. Use the `-m` flag to walk specific microservices: `repowalk -m auth,payment > context.md`

**Q: Does it actually redact secrets?**
A: Yes. Built-in regex patterns catch common secrets (API keys, tokens, passwords) and replace them with `[REDACTED]`. Use `--no-redact` to disable if needed.

**Q: Can I use this in CI/CD pipelines?**
A: Yes. RepoWalk is headless-friendly and works great in scripts and automation. Use `repowalk diff` to generate contexts for changed files only.

**Q: What token counter does it use?**
A: RepoWalk uses `tiktoken` (the same tokenizer OpenAI uses for GPT models), so token counts are accurate for Claude, ChatGPT, and compatible LLMs.

**Q: Can I customize the output format?**
A: Yes. Use `--json`, `--yaml`, or `--template` to export in your preferred format.

## 🤝 Contributing

We welcome all contributions! Please check out our [Contributing Guidelines](CONTRIBUTING.md) and our [Code of Conduct](CODE_OF_CONDUCT.md).

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

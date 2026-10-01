# RepoWalk 🚶‍♂️

[![NPM Version](https://img.shields.io/npm/v/repowalk.svg)](https://www.npmjs.com/package/repowalk)

RepoWalk is a blazing-fast, interactive CLI tool written in Go that walks your repository and packs it into a beautifully formatted Markdown document. 

Designed specifically for the era of AI coding, RepoWalk is the ultimate bridge between your local codebase and large language models (LLMs) like Claude, ChatGPT, or Ollama. Instead of copy-pasting individual files, you can generate a single "Super Context" file containing exactly what the AI needs to know.

<img width="1400" height="900" alt="demo" src="https://github.com/user-attachments/assets/c8642e59-1bb8-4fe4-bd75-a46ef0fc7ab7" />


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

## 🚀 Installation

**Homebrew (macOS / Linux):**
```bash
brew install qubeutils/tap/repowalk
```

**NPM (Node.js):**
Available on [npm](https://www.npmjs.com/package/repowalk)
```bash
npx repowalk ui
# OR globally
npm install -g repowalk
```

**Go Developer:**
```bash
go install github.com/QubeUtils/repowalk@latest
```

## 🎮 Usage

### 1. The Interactive TUI (Recommended)
Launch the beautiful terminal UI to visually select exactly what you want to include in your LLM context:

```bash
repowalk ui
```
*   `Up/Down/Left/Right` or `k/j/l`: Navigate the file tree (`Left` collapses folder, `Right` or `l` expands)
*   `Space`: Toggle select. Double-tap quickly for **Exclusive Select** (only includes that file/folder).
*   `e`: Enter **Edit Mode** to preview a file. Once inside, press `Space` to cycle through line states (Include, Exclude, Exclusively Include).
*   `Enter`: **Quick Save** the current context to your default output path.
*   `s`: **Save As...** interactive prompt to set a custom export filename and directory.
*   `c`: Copy the generated Markdown to your clipboard
*   `/`: Search and filter files (Supports multiple terms like `.go yaml`)
*   `m`: Toggle **Minimal UI** mode
*   `h` or `?`: Show Keyboard Shortcuts help screen
*   `PgUp/PgDn`: Scroll file preview
*   `q` or `Esc`: Quit

### 2. The CLI 
Run RepoWalk entirely headless to generate your context file immediately or manage your configuration.

**Basic Usage:**
```bash
# Output the current directory context to stdout
repowalk

# Redirect the output to a file
repowalk > repowalk_context.md

# Walk a specific directory
repowalk ./path/to/my/project > repowalk_context.md
```

**Commands:**

| Command | Description |
| :--- | :--- |
| `ui` | Launches an interactive TUI to visually browse the repository, select specific files/folders, and generate an LLM context dump. |
| `tree` | Prints a colored, visual tree structure of the repository, respecting `.gitignore`. Useful for previewing what will be included in the context dump. |
| `stats` | Calculates and displays statistics such as total files, total size, and estimated LLM token count for the current repository state. |
| `diff` | Creates an LLM context dump containing only the files that have changed according to the git state (untracked, unstaged, staged). |
| `init` | Creates a `.repowalk.json` file in your current directory. This local configuration overrides your global settings and can be committed to your VCS. |
| `completion` | Generates the autocompletion script for the specified shell. |
| `help` | Help about any command. |

**Global Flags (available for most commands):**

| Flag | Description |
| :--- | :--- |
| `--ignore-exts strings` | Override extensions to ignore (comma-separated). Overrides `~/.repowalk/config.json`. |
| `--json` | Export context as JSON format. |
| `--max-size int` | Maximum file size to include in bytes (default 1048576). |
| `-m, --microservice strings`| Selectively walk specific subdirectories (e.g., `-m auth,payment`). |
| `--no-redact` | Disable automatic secret redaction. |
| `--persona string` | Wrap context in a preset persona prompt (e.g., `security`, `refactor`, `review`). |
| `--template string` | Provide a custom template for the LLM dump. |
| `--yaml` | Export context as YAML format. |
| `-h, --help` | Show help for a command. |

**Tree Command Specific Flags:**

| Flag | Description |
| :--- | :--- |
| `--compact` | Print a compact representation of the tree. |
| `-L, --level int` | Descend only a certain number of directories deep. |

**Examples with Flags:**
```bash
# Use the Security persona and export as JSON
repowalk --persona security --json > context.json

# Walk only specific microservices, ignoring .md and .txt files
repowalk -m auth,payment --ignore-exts .md,.txt > context.md

# Preview the tree structure up to 2 levels deep
repowalk tree -L 2

# Output repo statistics
repowalk stats
```

## 🤝 Contributing

We welcome all contributions! Please check out our [Contributing Guidelines](CONTRIBUTING.md) and our [Code of Conduct](CODE_OF_CONDUCT.md).

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

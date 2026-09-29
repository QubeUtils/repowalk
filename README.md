# RepoWalk 🚶‍♂️

RepoWalk is a blazing-fast, interactive CLI tool written in Go that walks your repository and packs it into a beautifully formatted Markdown document. 

Designed specifically for the era of AI coding, RepoWalk is the ultimate bridge between your local codebase and large language models (LLMs) like Claude, ChatGPT, or Ollama. Instead of copy-pasting individual files, you can generate a single "Super Context" file containing exactly what the AI needs to know.

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

*More installation methods (Homebrew, APT) coming soon in v1.0!*

For now, you can install via `go install`:

```bash
go install github.com/QubeUtils/repowalk@latest
```

## 🎮 Usage

### 1. The Interactive TUI (Recommended)
Launch the beautiful terminal UI to visually select exactly what you want to include in your LLM context:

```bash
repowalk ui
```
*   `Up/Down/Left/Right` or `h/j/k/l`: Navigate the file tree
*   `Space`: Select/Deselect a file or folder
*   `e`: Enter **Edit Mode** to preview a file. Once inside, press `Space` to cycle through line states (Include, Exclude, Exclusively Include).
*   `Enter`: **Quick Save** the current context to your default output path.
*   `s`: **Save As...** interactive prompt to set a custom export filename and directory.
*   `c`: Copy the generated Markdown to your clipboard
*   `/`: Search and filter files
*   `q` or `Esc`: Quit

### 2. The CLI 
Run RepoWalk entirely headless to generate your context file immediately:

```bash
# Dump the current directory to repowalk_context.md
repowalk

# Output directly to stdout
repowalk -o stdout

# Automatically copy to clipboard
repowalk -c

# Use the Security persona
repowalk --persona security
```

## 🤝 Contributing

We welcome all contributions! Please check out our [Contributing Guidelines](CONTRIBUTING.md) and our [Code of Conduct](CODE_OF_CONDUCT.md).

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

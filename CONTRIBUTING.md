# Contributing to RepoWalk

First off, thank you for considering contributing to RepoWalk! It's people like you that make it such a great tool.

## Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/QubeUtils/repowalk.git
   cd repowalk
   ```

2. **Run the CLI:**
   ```bash
   go run . --help
   ```

3. **Run the TUI:**
   ```bash
   go run . ui
   ```

4. **Run the Tests:**
   ```bash
   go test ./...
   ```

## Pull Request Process

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes (`go test ./...`).
5. Ensure your code is properly formatted (`go fmt ./...`) and passes standard Go linting.
6. Issue that pull request!

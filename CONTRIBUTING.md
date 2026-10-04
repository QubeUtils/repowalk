# Contributing to RepoWalk

Thanks for thinking about contributing! We're glad you're here.

Whether you're fixing a bug, suggesting a feature, writing tests, improving docs, or just curious about how things work, we'd love your help. This guide should get you started.

## Getting Started

### What You'll Need

- Git
- Go (check `go.mod` for the version we're using)
- A terminal

### Set Up Your Dev Environment

Clone the repo and get it running:

```bash
git clone https://github.com/QubeUtils/repowalk.git
cd repowalk
go mod download
```

### Quick Commands

**See what the CLI can do:**
```bash
go run . --help
```

**Try the terminal UI:**
```bash
go run . ui
```

**Run the tests:**
```bash
go test ./...
```

**Format your code:**
```bash
gofmt -w .
```

## Ways You Can Help

- **Found a bug?** Report it—include steps to reproduce and what you expected to happen.
- **Have an idea?** Suggest it! We're open to features and improvements.
- **Want to write tests?** Always welcome.
- **See something in the docs that's unclear?** Fix it.
- **Ready to code?** Pick an issue or propose a change.
- **Like reviewing code?** We'd love your feedback on open PRs.

## Making Changes

### Create a Branch

```bash
git checkout -b feature/what-youre-doing
```

Keep it focused. One change per PR is easier to review.

### Write and Test

- Add or update tests if you're fixing a bug or adding a feature.
- Keep your code readable and idiomatic Go.
- If you're changing how the CLI or TUI works, update the docs too.

### Before You Push

```bash
go test ./...
gofmt -w .
```

Make sure everything passes.

### Open a Pull Request

When you're ready, push your branch and open a PR. In the description, tell us:

- What problem this solves (or what it adds)
- How you tested it
- Any notes about behavior changes or compatibility

That's it. We'll take it from there—thanks for the contribution!

## Code Style & Standards

- Write clear, idiomatic Go.
- Commit messages should be descriptive.
- Don't mix unrelated changes in one PR.
- Keep functions small and focused.
- Preserve existing behavior unless you're intentionally changing it.

## Questions?

Not sure where to start or how to approach something? Open an issue or discussion first. We're here to help.

Thanks for making RepoWalk better! 🎉

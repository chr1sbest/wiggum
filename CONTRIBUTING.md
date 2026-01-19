# Contributing to Wiggum

Thanks for your interest in contributing to Wiggum (Ralph)! This guide will help you get started.

## Quick Start (5 minutes)

### 1. Clone and Setup

```bash
git clone https://github.com/chr1sbest/wiggum.git
cd wiggum
```

### 2. Install Locally

```bash
make install
# Or directly: go install ./cmd/ralph
```

This builds and installs `ralph` to your `$GOPATH/bin` (which should be on your `PATH`).

### 3. Run Tests

```bash
make test
# Or directly: go test ./...
```

### 4. Format Code

```bash
make fmt
# Or directly: go fmt ./...
```

### 5. Run All CI Checks

Before submitting a PR, run:

```bash
make ci
```

This runs all tests and checks that code is properly formatted (same checks that CI runs).

## Development Workflow

### Using Make Commands

We provide a Makefile for common development tasks:

```bash
make help       # Show all available commands
make test       # Run all tests
make fmt        # Format all code
make lint       # Run linter (golangci-lint or go vet)
make build      # Build the ralph binary
make ci         # Run all CI checks locally
make install    # Install ralph to GOPATH/bin
make clean      # Remove build artifacts
```

### Running Tests

Run all tests before submitting a PR:

```bash
make test
# Or directly: go test ./...
```

For verbose output:

```bash
go test -v ./...
```

To run tests for a specific package:

```bash
go test ./internal/loop
```

### Code Style

We follow standard Go conventions:

- Run `make fmt` before committing
- Use `gofmt` style (tabs for indentation, standard formatting)
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Keep functions focused and reasonably sized
- Write clear, descriptive variable names

### Making Changes

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the code style guidelines

3. **Run tests** to ensure nothing broke:
   ```bash
   make test
   ```

4. **Format your code**:
   ```bash
   make fmt
   ```

5. **Run CI checks** to catch any issues:
   ```bash
   make ci
   ```

6. **Commit your changes** with a clear message:
   ```bash
   git commit -m "Add feature: description of what you did"
   ```

### Pull Request Process

1. **Push your branch** to GitHub:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Open a Pull Request** on GitHub targeting the `main` branch

3. **Describe your changes** in the PR:
   - What problem does this solve?
   - What approach did you take?
   - Are there any breaking changes?
   - Did you add or update tests?

4. **Wait for review** - maintainers will review your PR and may request changes

5. **Address feedback** by pushing new commits to your branch

6. **Merge** - once approved, a maintainer will merge your PR

## Project Structure

Understanding the codebase layout will help you navigate and contribute effectively:

```
cmd/ralph/           # CLI entry point and command definitions
internal/
├── loop/            # Loop execution engine
│   └── steps/       # Step implementations (agent, git-commit, etc.)
├── agent/           # Session and PRD management
├── banner/          # Welcome banner display
├── config/          # Configuration loading with hot-reload
├── tracker/         # Run state, metrics, and file locking
├── resilience/      # Circuit breaker and retry logic
├── status/          # Terminal UI progress display
├── eval/            # Evaluation framework
└── logger/          # Structured logging
configs/             # Default configuration templates
examples/            # Sample requirements files
evals/               # Evaluation suites and results
```

### Key Directories

- **`cmd/ralph/`** - Start here if you're adding a new command or modifying CLI behavior
- **`internal/loop/`** - Core loop orchestration and step execution logic
- **`internal/agent/`** - PRD parsing, task status management, and Claude Code session handling
- **`internal/config/`** - Configuration file loading and environment variable substitution
- **`configs/`** - Default configuration templates and prompts

## Testing

### Writing Tests

- Place test files next to the code they test (e.g., `loop.go` → `loop_test.go`)
- Use table-driven tests when testing multiple scenarios
- Test both success and error cases
- Use meaningful test names that describe what's being tested

Example:

```go
func TestParseTaskStatus(t *testing.T) {
    tests := []struct {
        name   string
        input  string
        want   TaskStatus
    }{
        {"todo status", "todo", StatusTodo},
        {"done status", "done", StatusDone},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ParseTaskStatus(tt.input)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Specific Tests

```bash
# Run tests in a specific package
go test ./internal/loop

# Run a specific test function
go test ./internal/loop -run TestExecuteLoop

# Run with race detector
go test -race ./...
```

## Dependencies

The project uses Go modules. To add a new dependency:

```bash
go get github.com/example/package
```

Then run `go mod tidy` to clean up:

```bash
go mod tidy
```

Commit both `go.mod` and `go.sum` when dependencies change.

## Documentation

- Update the README.md if you add new features or change user-facing behavior
- Add code comments for exported functions, types, and complex logic
- Keep comments concise and focused on "why" rather than "what"
- Update examples in `examples/` if you change requirements file format

## Getting Help

- Check existing [Issues](https://github.com/chr1sbest/wiggum/issues) for similar questions or problems
- Look for issues labeled `good first issue` if you're new to the project
- Open a new issue if you find a bug or have a feature request
- Be respectful and constructive in all interactions

## Code of Conduct

This project follows a Code of Conduct. By participating, you agree to uphold this code. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for details.

## License

By contributing to Wiggum, you agree that your contributions will be licensed under the MIT License. See [LICENSE](LICENSE) for details.

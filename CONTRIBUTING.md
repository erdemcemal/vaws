# Contributing to vaws

Thank you for your interest in contributing to vaws! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Make
- AWS CLI v2 (for testing)

### Setup

```bash
# Clone the repository
git clone https://github.com/erdemcemal/vaws.git
cd vaws

# Install dependencies
go mod download

# Build
make build

# Run
./bin/vaws
```

## Development Workflow

### Branch Naming

- `feature/` - New features (e.g., `feature/rds-support`)
- `fix/` - Bug fixes (e.g., `fix/tunnel-reconnect`)
- `docs/` - Documentation changes
- `refactor/` - Code refactoring

### Making Changes

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Run tests and linting
5. Commit with a descriptive message
6. Push and create a Pull Request

### Code Style

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test
```

### Commit Messages

Follow conventional commits format:

```
type(scope): description

[optional body]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `refactor` - Code refactoring
- `test` - Adding tests
- `chore` - Maintenance

**Examples:**
```
feat(api-gateway): add HTTP API support
fix(tunnel): handle reconnection on network change
docs(readme): add port forwarding guide
```

## Project Structure

```
vaws/
├── cmd/vaws/       # Application entry point
├── internal/
│   ├── app/            # Application initialization
│   ├── aws/            # AWS API clients
│   ├── config/         # Configuration management
│   ├── log/            # Logging
│   ├── model/          # Domain models
│   ├── state/          # Application state
│   ├── tunnel/         # Port forwarding logic
│   └── ui/             # Terminal UI components
│       ├── components/ # Reusable UI components
│       ├── layout/     # Layout management
│       └── theme/      # Colors and styling
├── assets/             # Screenshots and images
└── Makefile            # Build commands
```

## Adding New Features

### Adding a New AWS Resource Type

1. **Add AWS client methods** in `internal/aws/`:
   ```go
   // internal/aws/newresource.go
   func (c *Client) ListNewResources(ctx context.Context) ([]model.NewResource, error)
   ```

2. **Add model** in `internal/model/model.go`:
   ```go
   type NewResource struct {
       ID   string
       Name string
       // ...
   }
   ```

3. **Add state** in `internal/state/state.go`:
   ```go
   NewResources        []model.NewResource
   NewResourcesLoading bool
   NewResourcesError   error
   ```

4. **Add view** in `internal/state/state.go`:
   ```go
   const (
       // ...
       ViewNewResource View = iota
   )
   ```

5. **Add UI handling** in `internal/ui/ui.go`:
   - Add list component
   - Add load function
   - Add message handler
   - Add view rendering
   - Add keyboard navigation

### Adding New Keyboard Shortcuts

1. Add key binding in `internal/ui/keys.go`
2. Handle the key in `internal/ui/ui.go` Update method
3. Document in README.md

## Testing

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./internal/aws/...

# With coverage
go test -cover ./...
```

### Manual Testing

Test with your AWS account:

```bash
# Debug mode shows detailed logs
./bin/vaws --debug

# Test specific profile/region
./bin/vaws --profile test --region us-west-2
```

## Pull Request Guidelines

### Before Submitting

- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] Code is formatted (`make fmt`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated if needed

### PR Description

Include:
- What changed and why
- How to test the changes
- Screenshots for UI changes

### Review Process

1. Maintainer will review within a few days
2. Address any feedback
3. Once approved, PR will be merged

## Reporting Issues

### Bug Reports

Include:
- vaws version (`vaws --version`)
- Go version (`go version`)
- OS and terminal
- Steps to reproduce
- Expected vs actual behavior
- Error messages or logs

### Feature Requests

Include:
- Use case description
- Proposed solution (if any)
- Alternatives considered

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Questions?

Open an issue with the `question` label or start a discussion.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

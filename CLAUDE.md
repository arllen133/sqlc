.rules

## Code Style Guidelines

From `.agent/rules/standard.md`:
- Use short, focused functions with single responsibility
- Check and handle errors explicitly with wrapped errors (`fmt.Errorf("context: %w", err)`)
- Avoid global state; use constructor functions for dependency injection
- Leverage Go's context propagation for request-scoped values
- Write table-driven unit tests with parallel execution
- Document public functions with GoDoc-style comments

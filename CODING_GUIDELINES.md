# Coding Guidelines

Welcome to the HiddenLayer Azure APIM Guardrails coding guidelines! These guidelines are designed to ensure consistency and maintainability in our codebase. Please follow these guidelines when contributing to the project.

## General Principles

1. **Readability**: Write code that is easy to understand and maintain. Use meaningful variable names, comments, and formatting to enhance readability.

2. **Consistency**: Follow existing coding patterns and conventions within the codebase. Consistency makes it easier for developers to understand and contribute to the project.

3. **Simplicity**: Keep your code simple and straightforward. Avoid unnecessary complexity or over-engineering.

4. **Testing**: Write tests for your code whenever possible. Test-driven development (TDD) is encouraged to ensure code quality and reliability.

## Language-Specific Guidelines

### Go

1. **Formatting**: All Go code must be formatted with `gofmt`. Run `gofmt -s -w .` or use your editor's format-on-save.

2. **Naming**: Follow the [Go naming conventions](https://go.dev/doc/effective_go#names). Use MixedCaps for exported identifiers and mixedCaps for unexported ones. Avoid stuttering (e.g., prefer `policy.Fragment` over `policy.PolicyFragment`).

3. **Error Handling**: Always check returned errors. Wrap errors with context using `fmt.Errorf("doing X: %w", err)`. Do not discard errors silently.

4. **Packages**: Keep packages focused and cohesive. Avoid circular dependencies. Use `internal/` for packages that should not be imported by external consumers.

5. **Testing**: Write table-driven tests where appropriate. Place test files alongside the code they test (`*_test.go`). Use `go test ./...` to run the full suite.

## Version Control Guidelines

1. **Branching Strategy**: Follow a branching strategy such as GitFlow or a similar workflow for managing feature branches, releases, and hotfixes.

2. **Commit Messages**: Write clear and descriptive commit messages that explain the purpose of the changes. Use imperative mood and keep messages concise (< 72 characters if possible).

3. **Pull Requests**: Open pull requests for code review before merging changes into the main branch. Provide context, screenshots, or test cases where applicable.

## Code Review Process

1. **Code Reviews**: Participate in code reviews to provide feedback and ensure code quality. Reviewers should focus on readability, correctness, performance, and security.

2. **Timely Feedback**: Reviewers should provide timely feedback on pull requests to avoid delays in the development process.

3. **Continuous Improvement**: Learn from code reviews and incorporate feedback to improve coding practices and skills.

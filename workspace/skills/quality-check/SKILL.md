---
name: quality-check
description: Expert instructions for maintaining code quality, formatting, and project standards.
---

# Quality Check Skill

Expert instructions for ensuring all code changes meet project standards before being committed or pushed.

## Core Rules

1.  **Format First**: Always run `make fmt` (or `go fmt ./...`) before any commit. 
2.  **No Syntax Errors**: Never push code that hasn't been verified by the compiler or formatter.
3.  **Test Coverage**: Always run `go test ./...` to ensure no regressions were introduced. Aim for >85% coverage on new features.
4.  **Linters**: If available, run project-specific linters (e.g., `golangci-lint`).

## Pre-Commit Checklist

- [ ] Run `make fmt` to fix indentation and style.
- [ ] Verify there are no "missing ',' in argument list" or similar syntax errors.
- [ ] Run unit tests for the modified packages.
- [ ] Check overall project build: `go build ./...`.

## Workflow

1.  **Develop**: Apply changes to the codebase.
2.  **Verify**: Run `make fmt` and `go test`.
3.  **Correct**: If `make fmt` fails with a syntax error, locate the line and fix it before proceeding.
4.  **Submit**: Only commit once all quality checks pass.

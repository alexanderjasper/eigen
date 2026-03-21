# Plan: spec-cli/cmd-project — YAML validation + --all flag

## Overview

Two improvements to `eigen spec project`:
1. Pre-validate change files for backticks and bare colons before projecting
2. Add `--all` flag and `project-all` subcommand to reproject all modules at once

## Files to Modify

| File | Action |
|------|--------|
| `eigen/internal/spec/validation.go` | Add `LintError` type and `LintChangeFile` function |
| `eigen/cmd/spec_project.go` | Add `--all` flag, `project-all` subcommand, call linter before projection |

## Steps

**Step 1 — Add `LintError` and `LintChangeFile` to `eigen/internal/spec/validation.go`**

```go
type LintError struct {
    File    string
    Line    int
    Message string
}

func (e LintError) Error() string {
    return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
}

func LintChangeFile(filePath string, data []byte) []LintError
```

Line-by-line scan (not full YAML parse — errors may make YAML unparseable):
- Track block scalar state (`|`, `>` indicator lines → skip next lines)
- For unquoted scalar values (value doesn't start with `"`, `'`, `|`, `>`, `[`, `{`):
  - Flag any backtick character
  - Flag `: ` or trailing `:` in value portion

**Step 2 — Add `lintModule` helper in `eigen/cmd/spec_project.go`**

Reads all `.yaml` files from `storage.ChangesPath(specsRoot, path)`, calls `spec.LintChangeFile` per file, returns all errors.

**Step 3 — Refactor `runSpecProject` + add `--all` + `project-all`**

- Add `var projectAll bool`, register `specProjectCmd.Flags().BoolVar(&projectAll, "all", false, "Reproject all modules")`
- Add `specProjectAllCmd` with `Use: "project-all"`, registered on `specCmd`, delegates to same run function with `projectAll = true`
- `runSpecProject` logic:
  1. Determine scope: single path arg or all modules via `storage.WalkModules`
  2. Collect lint errors across all paths in scope
  3. If any lint errors: print all to stderr, return error (abort before any projection)
  4. For each path: call `reprojectModule(path)`
- No-arg + no `--all`: return error with usage hint

## AC Mapping

| AC | Step |
|----|------|
| AC-001: reprojects spec.yaml | Step 3 (single path flow) |
| AC-002: fails if module doesn't exist | Step 3 (check changes/ dir) |
| AC-003: reprojects all with --all or project-all | Step 3 |
| AC-004: aborts on backtick | Steps 1, 2, 3 |
| AC-005: aborts on bare colon | Steps 1, 2, 3 |
| AC-006: pre-validation aborts before any projection | Step 3 (collect all errors first) |
| AC-007: valid files pass silently | Steps 1, 3 |

## Build + Verification

```bash
cd eigen && go build ./...
go run . spec project spec-cli/cmd-project
go run . spec project --all
go run . spec project-all
go run . spec project nonexistent/module  # exit non-zero
go run . spec project                     # exit non-zero with usage hint
# Temporarily insert backtick in a change file, verify abort with file:line error
```

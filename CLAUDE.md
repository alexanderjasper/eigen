# Eigen — development guide

## Repo structure

```
eigen/          Go CLI source (go install . to build)
specs/          Eigen specifications for this repo (eigen is built using itself)
```

## Working with specs

### spec.yaml is generated — never edit it directly

`spec.yaml` in each module directory is projected from the `changes/` files by `eigen spec project`. Always edit change files; changes to `spec.yaml` will be overwritten.

The pre-commit hook re-runs projection automatically, but **does not stage the updated `spec.yaml` files**. After editing change files, you must explicitly stage and commit the projected `spec.yaml` alongside them:

```bash
git add specs/<module>/changes/001_foo.yaml specs/<module>/spec.yaml
git commit
```

Forgetting to commit `spec.yaml` means CI validates the stale projected file, not the one matching your change files.

### The `dependencies` field takes module paths only

`dependencies` in a change file (and the resulting `spec.yaml`) must contain **valid module paths** that resolve to real directories under `specs/`. Examples:

```yaml
dependencies:
  - spec-cli
  - spec-navigator/api
```

**Do not** put free-text descriptions here, even if they sound like requirements or environment prerequisites:

```yaml
# WRONG — these are not module paths and will fail eigen spec validate
dependencies:
  - approved plan (markdown)
  - git commands for commits
  - codebase with ability to edit/write files
```

If a module has no cross-module spec dependencies, use `dependencies: []`. Environment prerequisites (tools, capabilities, runtime context) belong in `description` or `behavior`, not in `dependencies`.

### Validating specs

```bash
eigen spec validate        # validates all modules
eigen spec validate <path> # validates a single module
```

CI runs this on every PR. A spec is invalid if any dependency entry does not resolve to a real module path.

### Change file status lifecycle

`draft` → `approved` → `compiled`

Only change files with status `compiled` are considered done. The compile-agent marks changes compiled via `eigen spec change-status`.

## Build

```bash
cd eigen
go build ./...
go test ./...
```

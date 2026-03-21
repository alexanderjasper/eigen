# Plan: spec-cli/cmd-scaffold — --force flag

## Overview

The `scaffold` command exists in `eigen/cmd/scaffold.go`. Add a `--force` / `-f` flag that removes existing skill and agent files before re-writing them, instead of erroring out.

## Files to Modify

- `eigen/cmd/scaffold.go` — only file that changes

## Steps

**Step 1 — Declare flag variable and register it**
Add `var scaffoldForce bool` and wire with `scaffoldCmd.Flags().BoolP("force", "f", false, "overwrite existing skill and agent files")` inside `func init()`.

**Step 2 — Make conflict check conditional**
Wrap the existing `if len(existing) > 0 { return error }` block in `if !scaffoldForce { ... }`.

**Step 3 — Remove existing files when --force is set**
After the conditional check, add a block that runs when `scaffoldForce` is true: iterate over skill paths and agent paths, call `os.Remove(p)` for each, swallow `os.ErrNotExist`. Do NOT touch `specsDir`.

**Step 4 — Write and print sections unchanged**
`MkdirAll` + `WriteFile` for skills/agents, `MkdirAll` for `specs/`, success print — no changes needed.

## AC Mapping

| AC | Step |
|----|------|
| AC-004: conflict detection without --force | Step 2 |
| AC-006: --force removes and rewrites | Steps 1, 3 |
| AC-007: --force leaves specs/ intact | Step 3 (explicitly skips specsDir) |

## Build + Verification

```bash
cd eigen && go build ./...
tmp=$(mktemp -d)
./eigen scaffold "$tmp"
./eigen scaffold "$tmp"           # must fail
./eigen scaffold --force "$tmp"   # must succeed
echo test > "$tmp/specs/keep.yaml"
./eigen scaffold --force "$tmp"
cat "$tmp/specs/keep.yaml"        # must still say "test"
```

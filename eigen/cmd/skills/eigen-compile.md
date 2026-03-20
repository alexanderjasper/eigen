---
name: eigen-compile
description: Implement (compile) an eigen spec into working code
---

Compile a spec into code for the eigen project. Specs are the source of truth — implement exactly what the spec says, no more, no less.

## Arguments
`/eigen-compile <module-path>`

## Workflow

1. **Read the spec**: `specs/<module-path>/spec.yaml`. This defines correctness.

2. **Read recent events**: `specs/<module-path>/events/` for context on the change.

3. **Explore the codebase**: understand existing patterns in `eigen/cmd/` and `eigen/internal/` before writing anything.

4. **Implement**: follow the cobra command pattern established in existing commands. The spec's acceptance criteria are your test cases.

5. **Build**: `cd eigen && go build ./...`

6. **Verify manually**: exercise each acceptance criterion from the spec against the built binary.

7. **Commit**: `feat(<domain>): implement <spec title>` — small atomic commits as you go.

## Constraints
- Implement exactly what the spec says — no extra features, no gold-plating
- If the spec is ambiguous or incomplete, **stop and report** — do not guess
- Do not modify spec files during compilation
- Follow the pattern: `func init() { specCmd.AddCommand(...) }`, `specsRoot` for path resolution, `storage` package for I/O

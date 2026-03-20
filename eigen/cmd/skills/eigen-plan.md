---
name: eigen-plan
description: Plan implementation of an eigen spec module
---

Plan the implementation of a spec for the eigen project.

## Arguments
`/eigen-plan <module-path>`

## Workflow

1. **Read the spec**: `specs/<module-path>/spec.yaml` — this is the source of truth.

2. **Read recent events**: the latest event file(s) in `specs/<module-path>/events/` for context on what changed and why.

3. **Explore existing patterns**: look at `eigen/cmd/` (cobra command files) and `eigen/internal/` (spec, storage packages) to understand what to reuse.

4. **Enter plan mode**: invoke `/plan` and design the implementation. The plan should include:
   - Files to create/modify with specific function signatures
   - How it fits the existing cobra command pattern (`init()` + `AddCommand`, `RunE`, `specsRoot`)
   - Each acceptance criterion mapped to a concrete implementation task
   - Build + manual verification steps

Do not implement anything — only plan.

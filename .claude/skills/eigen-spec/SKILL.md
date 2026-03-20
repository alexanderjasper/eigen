---
name: eigen-spec
description: Spec a feature by authoring events across one or more eigen modules
---

Spec a change for the eigen project. This could be a small single-event tweak to one module, or a large feature spanning multiple new and existing modules. Scale your effort to the scope — don't over-engineer a one-line fix, but don't under-spec a major feature.

## Arguments
`/eigen-spec [description]`

If the description is missing or too vague to proceed, ask the user what they want to build and why.

## Workflow

1. **Understand the change**: clarify scope with the user if needed. Identify which existing modules are affected and whether any new modules need to be created. For small changes this may be obvious immediately.

2. **Survey existing specs**: run `eigen spec list` and read the relevant `spec.yaml` files to understand current state before making changes. Skip modules that clearly aren't affected.

3. **Plan the spec changes**: decide which modules to create/update and what each event should contain. Think like an architect — the spec tree should reflect how a human thinks about the system, not how the code is structured. For a small change this is just one event in one module.

4. **For each new module**:
   - `eigen spec new <module-path>` to scaffold it
   - Edit the generated `001_initial.yaml` with the full initial spec
   - `eigen spec project <module-path>` to write spec.yaml

5. **For each existing module being updated**:
   - `eigen spec event <module-path>` to generate the next event template (prints path)
   - Edit the template — include only the fields that are changing
   - `eigen spec project <module-path>` to reproject

6. **Validate all changed modules**: `eigen spec validate`

7. **Commit**: one commit per logical spec unit: `spec(<module>): <summary>`

## Event file guidelines
- `summary`: one-line description of this event's change
- `reason`: *why* this change is being made — motivation, not restatement
- `type`: `created`, `updated`, or `removed`
- `changes`: only fields that are actually changing. Never include fields identical to current state.
- Acceptance criteria describe observable behavior (given/when/then), not implementation details

## Scope guidelines
- New top-level domain? Create a parent module first, then sub-modules
- Cross-module contracts (dependencies)? Update the `dependencies` field
- One event = one logical change; don't bundle unrelated changes into a single event

# Plan: ai-agent/skill-change — Naming & Named-Agent Convention Docs (AC-009, AC-010)

## Overview

Document two conventions that are already followed but not written down:
- AC-009: Spec module paths are domain-based (e.g. `ai-agent`), never sequence numbers. Only `changes/` filenames are numbered.
- AC-010: Subagents must be defined as `.claude/agents/<name>.md` with `name:` frontmatter. Use `subagent_type: <name>`. Never use `subagent_type: general-purpose` with inline prompts.

## Files to Modify

| File | Notes |
|------|-------|
| `.claude/agents/spec-agent.md` | Canonical — add Conventions section |
| `eigen/cmd/agents/spec-agent.md` | Embedded copy — mirror canonical |
| `.claude/agents/plan-agent.md` | Canonical — add conventions note |
| `eigen/cmd/agents/plan-agent.md` | Embedded copy — mirror canonical |
| `.claude/skills/eigen-spec/SKILL.md` | Canonical — add to Scope guidelines |
| `eigen/cmd/skills/eigen-spec.md` | Embedded copy — mirror canonical |

## Steps

**Step 1 — `.claude/agents/spec-agent.md`**: Add `### Conventions` section after Scope guidelines in Mode 1:

```markdown
### Conventions
- **Module naming** (AC-009): module paths use domain-based identifiers (e.g. `ai-agent`, `spec-cli/cmd-scaffold`), never sequence numbers. Only the YAML files inside `changes/` are numbered (e.g. `001_initial.yaml`).
- **Named agents** (AC-010): every subagent must be defined as `.claude/agents/<name>.md` with a `name:` frontmatter field. Invoke via `subagent_type: <name>`. Never use `subagent_type: general-purpose` with an inline prompt.
```

**Step 2 — `eigen/cmd/agents/spec-agent.md`**: Mirror Step 1 exactly.

**Step 3 — `.claude/agents/plan-agent.md`**: Add conventions note in the codebase exploration step:

```markdown
**Conventions to enforce in the plan:**
- Spec module paths must be domain-based (e.g. `spec-cli/cmd-new`), never sequence numbers (AC-009).
- Any agent invocations must reference named agent files via `subagent_type: <name>`, not inline prompts (AC-010).
```

**Step 4 — `eigen/cmd/agents/plan-agent.md`**: Mirror Step 3 exactly.

**Step 5 — `.claude/skills/eigen-spec/SKILL.md`**: Add to `## Scope guidelines`:

```markdown
- **Module naming**: module paths use domain-based identifiers (e.g. `ai-agent`, `spec-cli/cmd-scaffold`), never sequence numbers. Only `changes/` filenames are numbered.
- **Named agents**: subagents must be defined as `.claude/agents/<name>.md` with a `name:` frontmatter field and invoked via `subagent_type: <name>`. Never use `subagent_type: general-purpose` with an inline prompt.
```

**Step 6 — `eigen/cmd/skills/eigen-spec.md`**: Mirror Step 5 exactly.

**Step 7 — Build**: `cd eigen && go build ./...`

**Step 8 — Commit**: `feat(ai-agent): document spec naming and named-agent conventions (AC-009, AC-010)`

## Architectural Notes

- Changes are documentation-only (Markdown embedded as `[]byte`) — build verification confirms embed compiles cleanly.
- Both `.claude/` (live) and `eigen/cmd/` (embedded) copies must stay in sync — divergence would mean scaffolded projects have different rules.
- `compile-agent.md` and `eigen-change` SKILL.md are not changed — compile-agent never creates modules or invokes subagents; eigen-change already correctly demonstrates the named-agent pattern.

# Plan: Fix plan-agent architecture — read-only draft agent + eigen-change owns plan mode

## Context

Subagents cannot call `EnterPlanMode` / `ExitPlanMode` — those tools only work in the main conversation thread. The current `plan-agent.md` is specced to enter plan mode itself, which means the user never sees the plan UI and approval is never gated. The fix splits responsibility:

- **plan-agent** becomes a pure read-only research + draft agent: explores codebase, returns structured markdown plan as text, no plan mode, no file writes, no commits.
- **eigen-change** (main conversation skill) owns the full plan mode lifecycle: launches plan-agent, receives draft text, calls `EnterPlanMode`, presents draft, calls `ExitPlanMode`, writes `plan.md`, commits.

---

## Files to Modify

| File | Change |
|------|--------|
| `.claude/agents/plan-agent.md` | Full rewrite — remove plan mode, write, commit; add Constraints block |
| `eigen/cmd/agents/plan-agent.md` | Mirror of above (embedded copy for `eigen scaffold`) |
| `.claude/skills/eigen-change/SKILL.md` | Phase 2 rewrite — skill owns EnterPlanMode/ExitPlanMode, write, commit |
| `eigen/cmd/skills/eigen-change.md` | Mirror of above (embedded copy for `eigen scaffold`) |

No Go source files change. No new files created.

---

## Step-by-Step Plan

### Step 1 — Rewrite `.claude/agents/plan-agent.md`

- Update frontmatter `description` to: "Research codebase and return a draft implementation plan as text — read-only, no plan mode, no file writes"
- Remove `PLAN_OUTPUT_PATH` from the Inputs section
- Replace steps 3–5 (enter plan mode / write plan.md / commit) with a single "Return draft plan" step: format as structured markdown and return as text output
- Add explicit **Constraints** section:
  - Do not call `EnterPlanMode` or `ExitPlanMode`
  - Do not call `Write`, `Edit`, or any file-mutating tool
  - Do not run any `git` commands
  - Sole output is the draft plan text returned to eigen-change

### Step 2 — Mirror to `eigen/cmd/agents/plan-agent.md`

Copy Step 1 content verbatim (embedded source for `eigen scaffold`).

### Step 3 — Rewrite Phase 2 in `.claude/skills/eigen-change/SKILL.md`

Replace current Phase 2 block with:

1. Launch plan-agent with `SPEC_PATH`, `MODULE_PATH`, `BRANCH` only (no `PLAN_OUTPUT_PATH`), instructing it to return draft plan as text
2. Capture returned draft plan text
3. Call `EnterPlanMode` presenting the draft to the user
4. Call `ExitPlanMode` to collect approval
5. On approval: write draft to `.claude/plans/<BRANCH>/plan.md`, commit `plan(<module>): <summary>`, proceed to Phase 3
6. On rejection: `AskUserQuestion` for feedback → Spec Feedback Loop → restart Phase 2

Update the Notes section: "eigen-change enters plan mode after plan-agent returns the draft, so plan mode UI is presented in the main conversation thread."

### Step 4 — Mirror to `eigen/cmd/skills/eigen-change.md`

Copy Step 3 content verbatim.

### Step 5 — Build verification

```bash
cd eigen && go build ./...
```

---

## Verification

1. **Build**: `cd eigen && go build ./...` — must pass (embedded markdown, no logic changes)
2. **Audit plan-agent.md**: confirm no mention of `EnterPlanMode`, `ExitPlanMode`, `Write`, `Edit`, `git commit`, or `PLAN_OUTPUT_PATH`
3. **Audit eigen-change SKILL.md Phase 2**: confirm `EnterPlanMode`/`ExitPlanMode` present, plan-agent called with no `PLAN_OUTPUT_PATH`, file-write + commit logic present in skill after `ExitPlanMode`
4. **Sync check**: `.claude/agents/plan-agent.md` == `eigen/cmd/agents/plan-agent.md`; `.claude/skills/eigen-change/SKILL.md` == `eigen/cmd/skills/eigen-change.md`
5. **Scaffold smoke test** (optional): `eigen scaffold --force /tmp/test` → confirm `/tmp/test/.claude/agents/plan-agent.md` matches canonical

---
name: eigen-change
description: Orchestrate a full spec → plan → compile workflow using subagents with user approval gates
---

Implement a change for the eigen project through three phases: spec, plan, and compile. Each phase is handled by a specialized subagent. You review the output between phases and can provide feedback to refine before proceeding.

## Arguments
`/eigen-change <description>`

If description is missing, ask what the user wants to build.

---

## Setup

Before starting, collect context that all agents will need:

```bash
git branch --show-current
```

Store as `BRANCH`. The plan file path will be `.claude/plans/<BRANCH>/plan.md`.

Ask the user which module path to use (e.g. `ai-agent/skill-change`) or derive it from the description if obvious.

---

## Phase 1 — Spec

Launch the spec-agent to produce the spec:

```
Agent(
  subagent_type: spec-agent,
  prompt: |
    Mode: initial

    Feature description: <description from user>
    Module path: <module-path>

    Ask the user clarifying questions, explore the codebase, write the spec change files,
    run `eigen spec project <module-path>`, validate, and commit spec(<module>): <summary>.

    When done, report: the spec.yaml path and a one-line summary of what was specced.
)
```

After the agent completes:

1. Tell the user: "Spec phase complete. Review with `git diff HEAD~1` or in your editor."
2. Ask: **"Approve the spec? (yes / no + feedback)"**
   - **yes** → proceed to Phase 2
   - **no** → collect their feedback, then run **Spec Feedback Loop** below, then retry Phase 2

### Spec Feedback Loop

When the user rejects spec output, incorporate feedback before re-running:

```
Agent(
  subagent_type: spec-agent,
  prompt: |
    Mode: feedback

    Module path: <module-path>
    User feedback: <feedback text>

    Write a new change file (next sequence number) incorporating this feedback.
    Run `eigen spec project <module-path>`, validate, and commit
    spec(<module>): incorporate feedback on <aspect>.

    When done, report the change file written and what was updated.
)
```

After spec-agent commits the feedback change, re-show the approval prompt.

---

## Phase 2 — Plan

Launch the plan-agent to design the implementation:

```
Agent(
  subagent_type: plan-agent,
  prompt: |
    SPEC_PATH: specs/<module-path>/spec.yaml
    MODULE_PATH: <module-path>
    BRANCH: <BRANCH>
    PLAN_OUTPUT_PATH: .claude/plans/<BRANCH>/plan.md

    Read the spec, explore the codebase, enter plan mode for user review,
    write the approved plan to PLAN_OUTPUT_PATH, and commit plan(<module>): <summary>.

    Report the path to the written plan file when done.
)
```

The plan-agent will enter plan mode internally — you will see the plan mode UI for review and inline commenting. Plan mode approval/rejection is handled there.

After plan-agent completes and reports the plan file path:

1. Tell the user: "Plan phase complete. Plan written to `.claude/plans/<BRANCH>/plan.md`."
2. Ask: **"Proceed to implementation? (yes / no + feedback)"**
   - **yes** → proceed to Phase 3
   - **no** → collect feedback, run **Spec Feedback Loop** to update spec, then restart Phase 2

---

## Phase 3 — Compile

Launch the compile-agent to implement the code:

```
Agent(
  subagent_type: compile-agent,
  prompt: |
    SPEC_PATH: specs/<module-path>/spec.yaml
    MODULE_PATH: <module-path>
    PLAN_PATH: .claude/plans/<BRANCH>/plan.md

    Read the spec and plan completely before writing any code.
    Implement following the plan step by step.
    Build with `cd eigen && go build ./...`.
    Verify each acceptance criterion.
    Commit atomically: feat(<domain>): implement <title>.

    If spec or plan is ambiguous, stop and report — do not guess.
    Report what was implemented and any deviations from the plan when done.
)
```

After compile-agent completes:

1. Tell the user: "Implementation complete."
2. Ask: **"Looks good? (yes / no + feedback)"**
   - **yes** → done. Summarize: branch, spec path, plan path, commits made.
   - **no** → collect feedback, run **Spec Feedback Loop** to update spec, restart Phase 2 (re-plan from updated spec), then re-run Phase 3

---

## Notes

- **Spec is always updated first on rejection**: feedback becomes a new change file in `changes/` before re-running any phase. This ensures the spec stays authoritative — a fresh agent could re-plan or re-implement from spec.yaml alone.
- **Legacy skills remain**: `/eigen-spec`, `/eigen-plan`, `/eigen-compile` are still available for manual use.
- **Plan mode UI is preserved**: plan-agent uses Claude Code's built-in plan mode, so you get the familiar review UI with inline commenting.

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
2. Use AskUserQuestion to ask:
   - Question: "Approve the spec?"
   - Options: "Approve" (proceed to Phase 2), "Reject" (provide feedback to refine)
   - If rejected, prompt for feedback text via a follow-up AskUserQuestion, then run **Spec Feedback Loop** below, then re-show approval.

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

Launch the plan-agent to research the codebase and return a draft plan:

```
Agent(
  subagent_type: plan-agent,
  prompt: |
    SPEC_PATH: specs/<module-path>/spec.yaml
    MODULE_PATH: <module-path>
    BRANCH: <BRANCH>

    Read the spec, explore the codebase, and return a structured markdown draft plan
    as your text output. Do not write any files or make any commits.
)
```

Capture the returned draft plan text.

Call `EnterPlanMode` presenting the draft to the user for review.

Call `ExitPlanMode` to collect the approval decision.

On approval:
1. Write the draft to `.claude/plans/<BRANCH>/plan.md`
2. Commit: `plan(<module>): <one-line summary>`
3. Tell the user: "Plan phase complete. Plan written to `.claude/plans/<BRANCH>/plan.md`."
4. Use AskUserQuestion to ask:
   - Question: "Proceed to implementation?"
   - Options: "Proceed" (go to Phase 3), "Reject" (provide feedback to revise plan)
   - If rejected, prompt for feedback via follow-up AskUserQuestion, run **Spec Feedback Loop** to update spec, then restart Phase 2.

On rejection: use AskUserQuestion to collect feedback text, run **Spec Feedback Loop** to update the spec, then restart Phase 2.

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
2. Use AskUserQuestion to ask:
   - Question: "Implementation looks good?"
   - Options: "Approve" (done — summarize branch, spec path, plan path, commits made), "Reject" (provide feedback)
   - If rejected, prompt for feedback via follow-up AskUserQuestion, run **Spec Feedback Loop** to update spec, restart Phase 2, then re-run Phase 3.

---

## Notes

- **Spec is always updated first on rejection**: feedback becomes a new change file in `changes/` before re-running any phase. This ensures the spec stays authoritative — a fresh agent could re-plan or re-implement from spec.yaml alone.
- **Companion skills**: `/eigen-change-spec` and `/eigen-change-compile` are available for manual phase invocation.
- **Plan mode UI is preserved**: eigen-change enters plan mode after plan-agent returns the draft, so plan mode UI is presented in the main conversation thread.

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

Store as `BRANCH`.

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
    run `eigen spec project <module-path>`, and validate.

    When done, report: the spec.yaml path, the list of change file paths written (relative
    to the repo root, e.g. specs/<module-path>/changes/001_initial.yaml), and a one-line
    summary of what was specced.
)
```

After the agent completes:

1. Tell the user: "Spec phase complete. Review in your editor or with `git diff`."
2. Commit the spec files:
     git add specs/<module-path>/
     git commit -m "spec(<module>): <summary from spec-agent report>"

3. Tell the user: "Spec written. Open http://localhost:7171 to review and approve/reject in the browser. I'll proceed automatically when you submit."

   Start a background poll (run_in_background: true, timeout: 300000):
   ```bash
   MODULE="<module-path>"
   while true; do
     RESULT=$(curl -s "http://localhost:7171/api/modules/$MODULE/changes")
     # Check if all draft changes are now approved
     ALL_APPROVED=$(echo "$RESULT" | python3 -c "
   import sys,json
   changes=json.load(sys.stdin)
   draft=[c for c in changes if not c.get('status') or c['status']=='draft']
   print('yes' if len(draft)==0 else 'no')
   " 2>/dev/null)
     if [ "$ALL_APPROVED" = "yes" ]; then
       echo "APPROVED"
       echo "$RESULT"
       exit 0
     fi
     # Check if any change has a review_comment (rejection feedback)
     HAS_COMMENT=$(echo "$RESULT" | python3 -c "
   import sys,json
   changes=json.load(sys.stdin)
   comments=[c for c in changes if c.get('review_comment')]
   print('yes' if comments else 'no')
   " 2>/dev/null)
     if [ "$HAS_COMMENT" = "yes" ]; then
       echo "REJECTED"
       echo "$RESULT"
       exit 0
     fi
     sleep 3
   done
   ```
   Wait for the background task completion notification.

4. Read the decision from the background task output (first line is APPROVED or REJECTED):
   - "APPROVED":
       For each change file, run: `eigen spec change-status <module-path> <filename> approved`
       Commit: `chore(<module>): approve spec changes`
       Proceed to Phase 2.
   - "REJECTED":
       Extract review_comment values from the changes JSON:
       ```bash
       FEEDBACK=$(echo "$RESULT" | python3 -c "
       import sys,json
       changes=json.load(sys.stdin)
       lines=[f'{c[\"filename\"]}: {c[\"review_comment\"]}' for c in changes if c.get('review_comment')]
       print('\n'.join(lines) if lines else 'No specific comments provided.')
       ")
       ```
       Use `eigen spec change-comment <module-path> <filename> <comment>` to record
       the feedback if needed.
       Run Spec Feedback Loop with FEEDBACK as the user feedback text.
       Repeat from step 2.

### Spec Feedback Loop

When the user rejects spec output, incorporate feedback before re-running:

```
Agent(
  subagent_type: spec-agent,
  prompt: |
    Mode: feedback

    Module path: <module-path>
    User feedback: <feedback text>

    If the feedback indicates a fundamental rewrite (wrong framing, wrong direction, start over):
    - Delete all change files in specs/<module-path>/changes/ that have status="" or status="draft"
    - Write a fresh change file using the next available sequence number (may be 001 if the module
      is new, or higher if compiled/approved changes already exist)
    Otherwise:
    - Write a new change file at the next sequence number on top of existing ones

    Run `eigen spec project <module-path>` and validate.

    When done, report whether this was a rewrite or incremental, the change file written, and what was updated.
)
```

After spec-agent returns, commit all pending spec files:
  git add specs/<module-path>/
  git commit -m "spec(<module>): incorporate feedback on <aspect>"
(Use the aspect/summary from spec-agent's report.)
Then re-enter the review cycle (poll module changes from step 2).

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
1. Store the approved plan text from ExitPlanMode.
2. Tell the user: "Plan approved. Proceeding to implementation."
3. Proceed to Phase 3, passing the approved plan text inline to compile-agent.

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
    PLAN_CONTENT:
    <approved plan text>

    Read the spec and the PLAN_CONTENT above completely before writing any code. The plan text is provided inline — it is not a file.
    Implement following the plan step by step.
    Build with `go build ./...` from the `eigen/` subdirectory of the repo root.
    Verify each acceptance criterion.
    Commit atomically: feat(<domain>): implement <title>.

    Only compile changes whose status is `approved`; skip draft and compiled changes.
    The `eigen` CLI must be available in $PATH. If `eigen` is not found, stop and ask the user
    to install it first (`go install` from the `eigen/` directory).
    After successful build and commit, run `eigen spec change-status <module-path> <filename> compiled`
    for each compiled change file, then commit: `chore(<module>): mark changes compiled`.

    If spec or plan is ambiguous, stop and report — do not guess.
    Report what was implemented and any deviations from the plan when done.
)
```

After compile-agent completes:

1. Tell the user: "Implementation complete."
2. Use AskUserQuestion to ask:
   - Question: "Implementation looks good?"
   - Options: "Approve" (done — summarize branch, spec path, commits made), "Reject" (provide feedback)
   - If rejected, prompt for feedback via follow-up AskUserQuestion, run **Spec Feedback Loop** to update spec, restart Phase 2, then re-run Phase 3.

---

## Notes

- **Spec is always updated first on rejection**: feedback becomes a new change file in `changes/` before re-running any phase. This ensures the spec stays authoritative — a fresh agent could re-plan or re-implement from spec.yaml alone.
- **Companion skills**: `/eigen-change-spec` and `/eigen-change-compile` are available for manual phase invocation.
- **Plan mode UI is preserved**: eigen-change enters plan mode after plan-agent returns the draft, so plan mode UI is presented in the main conversation thread.

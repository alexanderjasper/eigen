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
    run `eigen spec project <module-path>`, validate, and commit spec(<module>): <summary>.

    When done, report: the spec.yaml path, the list of change file paths written (relative
    to the repo root, e.g. specs/<module-path>/changes/001_initial.yaml), and a one-line
    summary of what was specced.
)
```

After the agent completes:

1. Tell the user: "Spec phase complete. Review with `git diff HEAD~1` or in your editor."
2. Read each change file's raw YAML and POST the batch to the review API:
   Build a JSON array of change entries (change_id, file_path, change_yaml) for all
   change files reported by spec-agent.

   ```bash
   SESSION_ID=$(curl -s -X POST http://localhost:7171/api/reviews \
     -H 'Content-Type: application/json' \
     -d '{"module_path":"<module-path>","changes":[...]}' \
     | python3 -c "import sys,json; print(json.load(sys.stdin)['session_id'])" 2>/dev/null)
   ```

   If SESSION_ID is empty (server not running or curl failed): fall back to AskUserQuestion
   with "Approve the spec?" / "Approve" / "Reject" as before.

3. Tell the user: "Spec written. Open http://localhost:7171 to review and approve/reject in the browser."

4. Poll every 3 seconds until submitted:
   ```bash
   while true; do
     RESULT=$(curl -s http://localhost:7171/api/reviews/$SESSION_ID)
     STATUS=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])" 2>/dev/null)
     [ "$STATUS" = "submitted" ] && break
     sleep 3
   done
   ```

5. Read decision:
   ```bash
   DECISION=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['decision'])")
   ```
   - "approved":
       For each change file, run: `eigen spec change-status <module-path> <filename> approved`
       Commit: `chore(<module>): approve spec changes`
       Proceed to Phase 2.
   - "rejected":
       Extract change_comments from RESULT:
       ```bash
       FEEDBACK=$(echo "$RESULT" | python3 -c "
       import sys,json
       d=json.load(sys.stdin)
       lines=[f'{k}: {v}' for k,v in d.get('change_comments',{}).items() if v]
       print('\n'.join(lines) if lines else 'No specific comments provided.')
       ")
       ```
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
    Build with `cd eigen && go build ./...`.
    Verify each acceptance criterion.
    Commit atomically: feat(<domain>): implement <title>.

    Only compile changes whose status is `approved`; skip draft and compiled changes.
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

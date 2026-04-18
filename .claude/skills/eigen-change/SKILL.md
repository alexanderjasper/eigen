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

**Branch setup:**
- If the invocation args include `--branch <name>`, set `BRANCH=<name>` and skip branch creation.
- If the user explicitly states they want to continue on the current branch, run `git branch --show-current`, store as `BRANCH`, and skip branch creation.
- Otherwise, derive a branch name from the module path or description (replace `/` with `-`, prefix with `feat/`, e.g. `feat/skill-change-branch-from-main`), then:
  ```bash
  git fetch origin main
  git checkout -b <derived-branch-name> origin/main
  ```
  Store the new branch name as `BRANCH`.

Capture `WORKTREE`:
```bash
basename "$(git rev-parse --show-toplevel)"
```
Store as `WORKTREE`.

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
   eigen spec await-approval <module-path> --timeout 5m
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

Clear stale review_comment values before re-entering the poll. For each change file
that previously had a non-empty review_comment (the filenames extracted during the
rejection step), run:
  eigen spec change-comment <module-path> <filename> ""
This prevents old rejection comments from triggering a false positive on the next poll.

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
1. Derive PLAN_PATH:
   - Compute module slug: replace `/` with `-` in MODULE_PATH (e.g. `ai-agent/skill-change` → `ai-agent-skill-change`)
   - Compute UTC timestamp: `date -u +%Y%m%dT%H%M%SZ`
   - PLAN_PATH = `.eigen/plans/<module-slug>-<timestamp>.md`
2. Write the approved plan text to PLAN_PATH:
   ```bash
   mkdir -p .eigen/plans
   # Write the approved plan text returned by ExitPlanMode to PLAN_PATH
   ```
3. Tell the user: "Plan approved. Plan written to <PLAN_PATH>. Proceeding to implementation."
4. Proceed to Phase 3, passing PLAN_PATH to compile-agent.

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
    PLAN_PATH: <PLAN_PATH>

    Read the spec completely and then call Read on PLAN_PATH to load the approved plan before writing any code. If the file at PLAN_PATH does not exist, stop and report the missing path.
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

After compile-agent completes, proceed to Phase 4.

---

## Phase 4 — Review

Issue TWO parallel tool calls in the same message:

**Track 1 — review-agent subagent:**

```
Agent(
  subagent_type: review-agent,
  prompt: |
    SPEC_PATH: specs/<module-path>/spec.yaml
    MODULE_PATH: <module-path>

    Read the spec and compiled implementation. Before evaluating:
    1. Build using the project-appropriate command (e.g. `go build ./...` from `eigen/` for Go projects).
    2. If the change involves a server or HTTP API:
       - Start the server and poll until HTTP 200 (30s timeout).
       - Exercise relevant endpoints with curl and record responses.
    3. Run the project's test suite and capture output.

    Verify every AC. Label each LIVE or STATIC. Include "Track: AGENT" in the report header.
    Return the full compliance report.
)
```

**Track 2 — inline browser verification (main thread, same message as Track 1):**

In the same message as the Track 1 Agent call, use preview MCP tools inline:

1. Read the spec's behavior and AC descriptions for UI/web language ("page", "browser", "frontend", "UI", "button", "form", "route").
2. If a web UI or local server is applicable:
   a. Call `preview_start` with the appropriate URL (infer from spec; e.g. http://localhost:7171 for eigen serve).
   b. Call `preview_screenshot` — capture initial state.
   c. Call `preview_snapshot` — read rendered DOM, verify content.
   d. For ACs involving UI elements/interactions: `preview_click`, then `preview_screenshot`/`preview_snapshot` to verify state changes.
   e. Record which ACs were visually confirmed and any discrepancies. Label findings "Track: BROWSER".
3. If no web UI is applicable:
   - Record: "Track 2 — inline browser verification: no web UI applicable. Browser verification skipped."
   - This is not a failure.

**After both tracks complete, merge findings:**

- For each AC: combine results. An AC is PASS if either track confirms it (and neither finds a failure for an AC both tracks checked).
- Label each AC row with the verifying track(s): AGENT, BROWSER, or BOTH.
- Overall verdict: PASS only if all ACs pass in the combined result.
- Include a "Verification coverage" section listing which ACs each track covered.

Present the merged compliance report to the user.

Read the summary line from the combined report:

- **PASS** (all ACs pass):
    Use AskUserQuestion to ask:
    - Question: "Review passed. Approve to finish or reject to revise."
    - Options: "Approve", "Reject" (provide feedback)
    - If approved: create a GitHub PR:
      ```bash
      gh pr create --title "<module>: <change summary>" --body "$(cat <<'EOF'
      ## Summary
      - Spec: specs/<module-path>/spec.yaml
      - ACs implemented: <list AC IDs from spec>

      🤖 Generated with [Claude Code](https://claude.com/claude-code)
      EOF
      )"
      ```
      Capture the PR URL from stdout and present it to the user as the final output.
    - If rejected: prompt for feedback via follow-up AskUserQuestion, run **Spec Feedback Loop** to update spec, restart Phase 2, then re-run Phases 3 and 4.

- **PARTIAL or FAIL** (one or more ACs fail):
    Tell the user the review found issues and show the Issues section of the combined report.
    Use AskUserQuestion to ask:
    - Question: "Review found failing ACs. Re-compile to fix, or override and approve anyway?"
    - Options: "Re-compile" (pass review Issues as feedback into **Spec Feedback Loop**, restart Phase 2, re-run Phases 3 and 4), "Approve anyway" (create PR and note open issues in summary), "Reject" (provide additional feedback)

---

## Notes

- **Spec is always updated first on rejection**: feedback becomes a new change file in `changes/` before re-running any phase. This ensures the spec stays authoritative — a fresh agent could re-plan or re-implement from spec.yaml alone.
- **Companion skills**: `/eigen-change-spec`, `/eigen-change-compile`, and `/eigen-change-review` are available for manual phase invocation.
- **Plan mode UI is preserved**: eigen-change enters plan mode after plan-agent returns the draft, so plan mode UI is presented in the main conversation thread.

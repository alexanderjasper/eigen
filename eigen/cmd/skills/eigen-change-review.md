---
name: eigen-change-review
description: Run the implementation review phase to verify spec compliance after compile
---

Manually invoke the review phase for a compiled module. Runs two parallel tracks concurrently and merges findings into a single compliance report.

## Arguments
`/eigen-change-review <module-path>`

If module-path is missing, ask for it.

---

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
   a. Call `preview_start` with the appropriate URL (infer from spec).
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
- Include a "Verification coverage" section.

Present the merged compliance report to the user.

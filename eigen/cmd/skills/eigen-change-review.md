---
name: eigen-change-review
description: Run the implementation review phase to verify spec compliance after compile
---

Manually invoke the review phase for a compiled module. Launches the review-agent against the spec and presents the compliance report.

## Arguments
`/eigen-change-review <module-path>`

If module-path is missing, ask for it.

---

Launch the review-agent:

```
Agent(
  subagent_type: review-agent,
  prompt: |
    SPEC_PATH: specs/<module-path>/spec.yaml
    MODULE_PATH: <module-path>

    Read the spec and the compiled implementation. Before evaluating correctness:
    1. Build the binary: run `go build ./...` from the `eigen/` subdirectory of the repo root.
    2. If the change involves the spec-navigator, serve, or any HTTP API subsystem:
       - Start `eigen serve &` and wait for it to be ready (poll http://localhost:7171 until HTTP 200).
       - Exercise relevant API endpoints with curl (e.g. `curl -s http://localhost:7171/api/modules`).
       - Fetch or check relevant web pages where browser tools are available.
    3. For all other changes, still build the binary and run `go test ./...`.

    Verify every acceptance criterion. In the compliance report, clearly label each AC as
    verified through live execution (LIVE) or static analysis only (STATIC).

    Return the full compliance report.
)
```

Present the returned report to the user.

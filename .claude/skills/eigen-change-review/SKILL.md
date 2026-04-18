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

    Read the spec and the compiled implementation, verify every acceptance criterion,
    run the build/test suite, and return the full compliance report.
)
```

Present the returned report to the user.

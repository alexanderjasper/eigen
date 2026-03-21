# Multi-phase change workflow with subagent orchestration
**Branch**: 001-skill-workflow-design
**Created**: 2026-03-21
**Released**: false

Create a new `change` skill that orchestrates a three-phase workflow for feature development. When invoked, it launches specialized subagents: (1) spec subagent asks clarifying questions and produces spec in eigen-spec format (changes/ + spec.yaml), (2) plan subagent designs the implementation in markdown, (3) compile subagent implements the code. User reviews and approves after each phase.

## Rules
- Each phase is invoked via the Agent tool with a distinct subagent_type; all have full codebase access
- Specs and plans are written to files for user review via editor/git diff
- User approves via yes/no prompt after reviewing files; progression does not require explicit commit
- Planning phase inputs are approved specs; compile inputs are approved plans
- Each phase produces a small atomic commit (or set of commits)
- Subagent orchestration happens within the skill; no manual invocations required
- Skill is invoked with a feature description and optional short-name hint

## Out of scope
- Modifications to existing eigen-spec, eigen-plan, eigen-compile skills
- Integration with spec navigator UI or web-based workflows
- Scheduled or automated workflow triggers
- Multi-user collaboration or conflict resolution

## Affected areas
- `.claude/skills/` — new `change` skill definition with three-phase workflow
- `eigen/cmd/` — new `cmd-change` command that invokes the skill
- Agent orchestration — uses Agent tool with subagent_type for spec/plan/compile phases
- Skill distribution — embedded in `eigen scaffold` via existing mechanisms

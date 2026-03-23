# Eigen

> *An eigenvector expresses the fundamental nature of a linear transformation — unchanged in direction, only scaled. Eigen specifications express the fundamental nature of software: the essential properties that define what it is, stripped of implementation noise.*

Eigen is a framework for enterprise-grade specification-driven development, where **specifications are the source code** and AI models are the compiler.

---

## The Problem

Software development has an inversion problem.

We write code — a machine-readable artifact — and then struggle to keep human-readable documents (requirements, architecture diagrams, wikis, ADRs) in sync with it. The code is the truth. Everything else drifts. New engineers read the code. Decisions live in Slack threads. The "documentation" is archaeology.

AI coding tools have made this worse in a subtle way. They accelerate code production without addressing the underlying problem. The result is vibe coding at scale: faster drift, faster accumulation of decisions that nobody recorded, faster accrual of a codebase that the team understands less and less over time. Studies show AI-generated code carries significantly higher rates of security vulnerabilities and major defects when produced without structured specification. The velocity is real. So is the chaos.

Eigen inverts this.

---

## The Vision

In Eigen, **specifications are the source of truth**. Code is a compiled artifact — authoritative for the machine, but not for the human. You do not read the code to understand the system. You read the specifications.

The workflow is:

1. A developer describes the desired behavior; an AI spec agent authors the change.
2. A human reviews and approves the specification change.
3. An AI model compiles the specification into working software.
4. Acceptance criteria derived from the specification verify correctness.
5. The software ships.

Nowhere in this workflow does a human need to read implementation code. This is the north star. We are honest that we will not reach it on day one — but every decision we make should be in service of it.

---

## Core Principles

### 1. Specifications as Source Code

Specifications are not documentation. They are not a guide for developers who will then make their own decisions. They are the source. The AI model that compiles them is not an assistant — it is a compiler. Like any compiler, it should produce deterministic, correct output from valid input without requiring clarification.

This means specifications must be **exhaustive for everything that matters**: behavior, performance characteristics, data models, user-facing contracts, security requirements, acceptance criteria. Implementation details that do not materially affect these outcomes are left to the compiler's discretion.

### 2. The Simplicity Constraint

The failure mode of every previous attempt at specification-driven development (Model-Driven Architecture being the canonical cautionary tale) was that the specifications became as complex as the code. A specification system that requires specialists to write and read is not a specification system — it is just another codebase written in a different language.

Eigen enforces a hard constraint: **a single specification node should not exceed what a competent engineer can read and fully grasp in a few minutes**. When a specification grows beyond this, it must be decomposed. This is a design principle, not a guideline.

### 3. Domain-First Hierarchy

Specifications are organized by domain and feature — the way humans think about software — not by architectural layer. A user does not experience a "data layer." They experience checkout, or messaging, or reporting.

Technical concerns (data models, APIs, infrastructure) are expressed within domain modules, not as top-level architectural axes. Cross-cutting concerns (authentication, logging, error handling, observability) live in dedicated cross-domain modules that other specifications can reference but do not need to restate.

### 4. Change-Sourced Specification History

Specifications are never edited in place. A specification change is appended as an immutable **change** — a semantic, human-readable description of what changed and why. The **current specification state** is a projection of all accumulated changes.

This gives Eigen several properties that matter at enterprise scale:

- The full rationale for every decision is preserved, not just the outcome.
- You can reconstruct the state of specifications at any point in history.
- When working on a change, a developer can inspect recent changes to understand the context they are working within.
- Specification history and code history are related but independent — a single logical change produces a spec change and a corresponding code commit, linked but separately legible.

This is directly inspired by change sourcing in software architecture. The insight is that the same pattern that makes change-sourced systems auditable and reversible applies equally well to the specifications that describe them.

### 5. Minimal Technology Commitment

Specifications should express the minimum technology constraints necessary to produce correct, performant software. Technology choices should be explicit enough to be binding, but sparse enough that the target platform could be changed by modifying a small number of specification nodes rather than the entire tree.

The AI compiler makes technology choices for everything not specified. Those choices should be idiomatic, current, and consistent — but they are implementation detail, not specification truth.

### 6. Verification from Specification

Acceptance criteria are a first-class component of specifications. They define what "compiled correctly" means. The AI compiler generates tests from these criteria as part of the compilation process. A compilation is not complete until the generated tests pass.

Acceptance criteria should be expressed in terms of observable behavior — inputs, outputs, side effects — not implementation internals. The goal is specifications that a non-engineer stakeholder could read and confirm describe the right behavior, even if the technical framing requires some translation.

We are deliberately cautious about how much formal structure to impose on acceptance criteria. Formalism prevents ambiguity but introduces bloat. We will discover the right balance through practice.

---

## The Compilation Process

When a specification changes, the following occurs:

1. **Spec authoring**: A developer describes desired behavior; an AI spec agent authors the change file and reprojects the current spec state.
2. **Spec review**: The developer reviews the change and the resulting projection in the browser UI. This is the primary human review step — reviewing intent, not implementation. Feedback triggers a new change file rather than editing in place.
3. **Planning**: The AI model explores the codebase and produces an implementation plan mapped to the spec. The developer reviews and approves the plan before any code is written.
4. **Compilation**: The AI model reads the full specification projection and implements it, verifying each acceptance criterion.
5. **Commit**: Passing compilation produces a linked pair: a spec commit and a code commit. Change files are marked `compiled`.

The AI model must be able to complete step 4 **without asking for clarification**. If it cannot, the specification is incomplete. Incompleteness is a specification defect, not an invitation for the AI to make assumptions about things that matter.

---

## Scale and Teams

Eigen is designed for teams of 10–50 engineers working on software that spans multiple services, domains, and potentially deployment targets. Several properties are necessary at this scale that are not necessary for solo or small-team work:

- **Specification ownership**: Modules have owners. Changes to a module require review from its owner.
- **Cross-module contracts**: When one module depends on behavior defined in another, that dependency is explicit in the specification — not inferred from the code.
- **Parallel compilation**: Independent modules can be compiled in parallel. The specification hierarchy defines the dependency graph that makes this safe.
- **Conflict detection**: If two pending changes affect overlapping parts of the specification, this is a merge conflict at the spec level, resolved before compilation begins.

---

## What We Do Not Know Yet

Eigen is a framework in formation. There are open questions we are committed to answering through building, not through speculation:

- **Acceptance criteria formalism**: The current format (given/when/then per criterion) is working but the right level of structure for complex behavior is still being discovered through practice.
- **Cross-cutting concern placement**: The right structure for concerns like authentication, observability, and error handling that touch every domain is not yet settled.
- **Developer tooling**: The spec navigator (`eigen serve`) covers browsing and change review. Deeper tooling — cross-module dependency graphs, specification history visualization, conflict detection — is not yet built.
- **Existing codebase adoption**: Eigen's eventual goal includes the ability to reverse-compile an existing codebase into specifications — effectively translating legacy code into Eigen's source of truth. This is not a near-term objective, but it informs design decisions we make now.

---

## Current Scope

The initial implementation of Eigen targets:

- Greenfield software development
- Teams using Claude Code as the AI compilation engine
- Software delivered as one or more networked services
- Engineers as specification authors (product and domain experts are a future evolution)

---

## What Eigen Is Not

- **Eigen is not a no-code platform.** Writing specifications requires engineering judgment. The goal is to eliminate the need to read and write implementation code, not to eliminate the need for engineering skill.
- **Eigen is not prompt engineering.** Specifications are not prompts. They are structured, versioned, owned artifacts that express system truth independent of any particular AI model.
- **Eigen is not documentation generation.** Specifications are not generated from code. They precede code. Code is generated from them.
- **Eigen is not a testing framework.** Acceptance criteria are part of specifications, but Eigen's primary concern is the correctness and completeness of specifications, not the mechanics of test execution.

---

## Getting Started

### Install

```bash
git clone https://github.com/alexanderjasper/eigen
cd eigen/eigen
go install .
```

### Set up a new project

Run `eigen scaffold` in your project root to install the Claude skills, subagent definitions, and create the `specs/` directory:

```bash
cd my-project
eigen scaffold
```

This creates:
- `.claude/skills/eigen-change/SKILL.md`
- `.claude/skills/eigen-change-spec/SKILL.md`
- `.claude/skills/eigen-change-compile/SKILL.md`
- `.claude/agents/spec-agent.md`
- `.claude/agents/plan-agent.md`
- `.claude/agents/compile-agent.md`
- `specs/`

Use `eigen scaffold --force` to overwrite existing skill and agent files.

> If your specs live somewhere other than a `specs/` directory above your CWD, set `EIGEN_SPECS=<path>` or pass `--specs <path>` to any command. The CLI also walks up from the CWD automatically, so you can run commands from any subdirectory of your project.

---

## The Development Workflow

Eigen development is driven by Claude Code skills. The primary skill orchestrates the full workflow; companion skills let you invoke individual phases manually.

### Primary workflow — `/eigen-change`

```
/eigen-change <description of what you want to build>
```

This skill runs the full spec → plan → compile pipeline through three phases:

**Phase 1 — Spec**: A spec subagent writes change files for the relevant module, projects the current spec state, and validates. The spec is then submitted for review via the browser UI (see `eigen serve` below). You can approve or reject with per-change comments. On rejection, feedback is incorporated as a new change file and re-submitted.

**Phase 2 — Plan**: A plan subagent reads the spec and explores the codebase, then returns a structured implementation plan. This is presented in plan mode for your review before any code is written.

**Phase 3 — Compile**: A compile subagent implements exactly what the spec says — no more, no less. Each acceptance criterion is treated as a test case. If the spec is ambiguous, compilation stops and reports the gap rather than guessing. Implemented changes are marked `compiled` in their change files.

### Companion skills

`/eigen-change-spec <module-path>` — invoke only the spec phase manually.

`/eigen-change-compile <module-path>` — invoke only the compile phase manually.

### Spec review UI — `eigen serve`

```bash
eigen serve
```

Starts a local web UI at `http://localhost:7171` for browsing specs and reviewing pending change submissions. The `/eigen-change` skill posts change batches to this server for approval — if the server is not running, it falls back to an in-conversation approval prompt.

### Change lifecycle

Each change file has a status that tracks its progress through the workflow:

- `draft` — written but not yet approved
- `approved` — approved in the review UI; ready for compilation
- `compiled` — implemented and committed

---

## CLI Reference

```
eigen spec list [prefix]                        List all spec modules
eigen spec new <path>                           Create a new spec module
eigen spec show <path>                          Print the current spec projection
eigen spec change <path> [--edit]               Record a new change (--edit opens $EDITOR)
eigen spec project [path]                       Reproject spec.yaml from changes
eigen spec validate [path]                      Validate completeness and dependencies
eigen spec change-status <path> <file> <status> Set change status (draft|approved|compiled)

eigen serve [--port 7171] [--no-open]           Browse specs and review changes in a web UI
eigen scaffold [path] [--force]                 Initialize a new project
```

---

## Status

Early but functional. The CLI, spec format, AI workflow skills, and spec navigator UI are built and in use. The framework is being developed using itself. Open questions are being answered through practice.

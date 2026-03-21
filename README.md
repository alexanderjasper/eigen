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

1. A developer writes or modifies a specification.
2. A human reviews and approves the specification change.
3. An AI model compiles the specification into working software.
4. Tests derived from the specification verify correctness.
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

1. **Spec authoring**: A developer writes a change describing the new or modified behavior.
2. **Spec review**: Another developer (or the same developer after reflection) reviews the change and the resulting projection. This is the primary human review step — reviewing intent, not implementation.
3. **Compilation**: The AI model reads the full specification projection (and optionally the recent changes for context) and produces code changes.
4. **Verification**: Generated tests derived from acceptance criteria run. Additional static analysis and security scanning run.
5. **Commit**: Passing compilation produces a linked pair: a spec commit and a code commit.

The AI model must be able to complete step 3 **without asking for clarification**. If it cannot, the specification is incomplete. Incompleteness is a specification defect, not an invitation for the AI to make assumptions about things that matter.

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

- **Specification format**: We do not yet have a definitive answer on whether specifications should be structured prose, a lightweight DSL, YAML/TOML with natural language fields, or something else. The constraint is that the format must be human-readable without tooling, and machine-parseable without ambiguity. We will find this through iteration.
- **Acceptance criteria formalism**: How much structure is the right amount for expressing testable behavior without creating specification bloat? Unknown.
- **Cross-cutting concern placement**: The right structure for concerns like authentication, observability, and error handling that touch every domain is not yet settled.
- **Developer tooling**: Navigating a large specification tree, understanding cross-module dependencies, and inspecting specification history all require purpose-built tooling. We know what it needs to do; we have not yet built it.
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

Run `eigen scaffold` in your project root to install the Claude skills and create the `specs/` directory:

```bash
cd my-project
eigen scaffold
```

This creates:
- `.claude/skills/eigen-spec/SKILL.md`
- `.claude/skills/eigen-plan/SKILL.md`
- `.claude/skills/eigen-compile/SKILL.md`
- `specs/`

> If your specs live somewhere other than a `specs/` directory above your CWD, set `EIGEN_SPECS=<path>` or pass `--specs <path>` to any command.

---

## The Development Workflow

Eigen development happens in three steps, each driven by a Claude Code skill.

### 1. Author a spec — `/eigen-spec`

```
/eigen-spec [description of what you want to build]
```

Claude will identify which spec modules are affected, create or update them by writing changes, project the current state, and validate. Each change records not just *what* changed but *why*.

After this step you have a reviewed, validated spec. That is the human review checkpoint — you are approving intent, not implementation.

### 2. Plan the implementation — `/eigen-plan`

```
/eigen-plan <module-path>
```

Claude reads the spec and recent changes, explores the existing codebase, and produces a detailed implementation plan mapped to each acceptance criterion. No code is written yet.

Use this step when the change is non-trivial and you want to align on approach before compilation begins.

### 3. Compile the spec to code — `/eigen-compile`

```
/eigen-compile <module-path>
```

Claude implements exactly what the spec says — no more, no less. Each acceptance criterion in the spec is treated as a test case. If the spec is ambiguous or incomplete, compilation stops and reports the gap rather than guessing.

---

## CLI Reference

```
eigen spec list [prefix]          List all spec modules
eigen spec new <path>             Create a new spec module
eigen spec show <path>            Print the current spec projection
eigen spec change <path>          Record a new change
eigen spec project [path]         Reproject spec.yaml from changes
eigen spec validate [path]        Validate completeness and dependencies

eigen serve [--port 7171] [--open]   Browse specs in a web UI
eigen scaffold [path]             Initialize a new project
```

---

## Status

Early formation. The manifesto precedes the framework. Both will evolve.

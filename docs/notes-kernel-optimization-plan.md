---
title: Kernel Optimization Plan
required: false
sop:
  - Read this doc when planning kernel optimization work or reviewing runtime tradeoffs.
  - Update this doc when kernel priorities, sequencing, or validation gates change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Kernel Optimization Plan

## Status

This document is the current canonical strategy for kernel optimization work in `simsh`.

Use it as the strategic source of truth for:
- what the kernel should optimize for;
- what the kernel should not optimize for;
- how workstreams should be sequenced;
- how lower-level execution backlog items should be judged.

It is intentionally not the execution backlog itself. Concrete work items, validation commands, and done gates belong in `docs/notes-kernel-execution-backlog.md`.

The current execution route for backlog items is the feat/task harness under `.bagakit/ft-harness/feats/`.

## Goal
Make `simsh` a lightweight execution kernel that agents can trust as a working environment without paying the complexity cost of a full VM, container stack, or POSIX shell.

The target property is not "shell completeness". The target property is:
- predictable execution;
- trustworthy filesystem semantics;
- structured, planner-friendly side effects;
- low enough runtime cost to make background delegation cheap.

## Optimization Thesis

The project should optimize `simsh` as an agent-native runtime kernel, not as:
- a smaller Bash clone;
- a CLI-first product;
- an HTTP-first product;
- a GUI computer-use environment;
- a domain-specific workflow engine.

The strongest external signals currently point in the same direction:
- real file workflows matter;
- file editing and shell execution are the highest-value primitives;
- interface quality matters as much as model quality;
- observability, reviewability, and isolation become increasingly important as agents scale.

## Scope

This plan is for kernel work only:
- `pkg/contract`
- `pkg/sh`
- `pkg/fs`
- `pkg/engine/runtime`
- kernel-adjacent filesystem implementations where they define default runtime behavior

This plan intentionally defers entry-adapter optimization:
- `pkg/cmd`
- `pkg/service/httpapi`
- `cmd/simsh-cli`
- `cmd/simshd`

Adapter-specific product logic is also out of scope except where it pressure-tests kernel contracts.

## Non-Goals

- Full POSIX compatibility
- GUI automation or desktop-environment realism
- Product-specific memory semantics in core packages
- Expanding command breadth before kernel invariants are hardened
- Letting one reference integration dictate core abstractions prematurely

## Current Pressure Points

These are the main areas where the current kernel needs to improve:

### 1. Boundary trust
- Path access/capability claims are not yet strong enough to be treated as hard truth in all cases.
- Real-path escape handling in default filesystem implementations needs to be tightened.

### 2. Trace fidelity
- `ExecutionTrace` is directionally right but does not yet capture every important mutation/seam with enough precision to be considered full SSOT.

### 3. Path resolution model
- The kernel currently leans heavily on absolute paths and a fixed root view, which is safe but not yet natural enough for agent work.
- The project still lacks a fully explicit model for session-local `cwd`, relative path resolution, and how those semantics interact with mounts and capability restrictions.

### 4. External seam completeness
- The `/bin` external command seam is still too text-first and loses structured detail that planners and reviewers would want.

### 5. Session and cancellation realism
- Sessions exist, but the broader kernel model for long-running, isolated, and interruptible work is not yet fully hardened.

## Workstreams

### P0. Boundary and Capability Trust
This is the top priority. Until this is solid, every higher-level abstraction is suspect.

Objectives:
- Ensure path writability, removability, and mutability claims are true in practice.
- Eliminate path escape classes in default filesystem implementations.
- Make mount-backed and synthetic paths uniformly immutable through all mutation routes.
- Ensure composite commands (`cp`, `mv`, `rm`, `mkdir`, `touch`, redirections, edit flows) cannot create partial side effects because of inconsistent preflight semantics.

Key actions:
- Tighten real-path resolution and descendant handling in default filesystem adapters.
- Add regression tests for symlink escape, nested missing-parent escape, and mount/synthetic path mutation attempts.
- Review `CheckPathOp` semantics and make preflight guarantees explicit and consistent.

Exit criteria:
- No known path-escape write/remove flows remain in default runtime filesystems.
- Capability metadata matches actual enforcement behavior.
- Composite mutation commands either succeed fully or fail before mutation begins when preflight should reject them.

### P1. Agent-Native Action Surface
The kernel should expose the smallest action surface that gives agents the most leverage.

Objectives:
- Keep file editing, repository navigation, search, and shell execution reliable and low-ambiguity.
- Prefer commands and feedback patterns that reduce model guessing.
- Keep errors classifiable and stable enough for recovery policies.
- Introduce a path resolution model that makes file-system use feel natural without weakening the capability model.

Key actions:
- Review builtin command set through an ACI lens instead of a shell-completeness lens.
- Standardize error categories for path, policy, syntax, unsupported, and environment failures.
- Check whether a few kernel-level command refinements would remove common agent mistakes without broadening the shell surface too much.
- Design an explicit session-local virtual `cwd` model rather than implicitly inheriting host shell semantics.
- Introduce a unified path resolution layer so relative paths resolve into normalized virtual absolute paths before capability enforcement.
- Distinguish shell-relative paths from document-relative or skill-relative references so higher-level tooling can keep owning non-shell reference semantics where appropriate.
- Ensure mount-backed and synthetic paths remain capability-limited even when reached through future relative-path flows.

Exit criteria:
- The common software-work loop (`inspect -> search -> edit -> test -> verify`) can be expressed cleanly.
- Errors for common failure classes are stable and easy to branch on.
- Relative-path support, if enabled, is explicit, traceable, session-scoped, and does not blur path reachability with path mutability.

### P2. Structured Result and Trace Completeness
`ExecutionResult` and `ExecutionTrace` should be treated as first-class kernel outputs.

Objectives:
- Make result/trace the machine-consumable SSOT for execution outcomes.
- Capture enough detail that planners, reviewers, and adapter layers do not need to scrape free-form stdout for basic facts.
- Preserve mutation, denial, and resource-usage truth across builtins and external seams.

Key actions:
- Audit bytes-read/bytes-written accounting for real mutations.
- Improve separation and completeness of stdout/stderr/error reporting where the current external seam loses signal.
- Review whether trace should eventually include richer causality hooks without polluting the core with product semantics.

Exit criteria:
- File operation traces are faithful enough to support planning and review without hidden heuristics.
- External command integration no longer collapses important result structure by default.

### P3. Session, Isolation, and Interruptibility
Once the kernel is trustworthy for one-shot work, it should become trustworthy for longer-lived work.

Objectives:
- Make session semantics predictable.
- Keep policy ceiling and narrowing rules explicit.
- Improve runtime behavior for long-running and concurrently scoped agent work.
- Make timeout/cancel semantics meaningful in the actual filesystem and execution loops, not only present in policy structs.

Key actions:
- Audit context use in filesystem and execution paths.
- Tighten lifecycle semantics for create/checkpoint/resume/close.
- Identify what kernel support is required for safe parallel sessions without turning core into an orchestrator.

Exit criteria:
- Session state is durable where intended and absent where not intended.
- Timeout/cancel behavior is observable and effective enough to be relied on.
- Isolation assumptions for concurrent agent work are explicit and testable.

### P4. Reference Validation and Metrics
Kernel quality should be proven with workloads, not only argued from interfaces.

Objectives:
- Validate kernel abstractions against a small set of realistic agent tasks.
- Measure whether the runtime improves delegation and review loops, not only whether unit tests pass.

Key actions:
- Build a small `simsh`-native benchmark focused on file-workflow tasks.
- Add contract tests for adapter-backed validation where needed.
- Track outcome metrics that matter to agent users.

Suggested metrics:
- time to reviewable patch
- trace completeness for file operations
- patch correction loop count
- session success rate for well-scoped tasks
- number of tasks completed asynchronously without step-by-step supervision

Exit criteria:
- At least one reference workload demonstrates that the kernel abstractions improve real agent work rather than only unit-test cleanliness.

## Sequencing

Recommended order:

1. P0 boundary and capability trust
2. P1 path resolution and virtual `cwd` design
3. P2 result/trace completeness for the most important file and shell seams
4. Remaining P1 action-surface cleanup
5. P3 session/isolation/interruptibility
6. P4 reference validation and metric tracking

This order is intentional:
- a broader action surface is dangerous if the kernel contract is not trustworthy;
- path resolution should become natural only after boundary rules are hard enough to trust;
- trace work should follow closely because relative-path and `cwd` semantics need machine-visible execution context;
- a session model is hard to evaluate if side effects are not faithfully reported;
- benchmark and product-layer evaluation should come after the kernel stops leaking abstraction debt.

## Decision Rules

When choosing whether to add something to kernel:
- add it only if it improves determinism, trust, or agent leverage across workloads;
- do not add it only because it is convenient for one entry adapter or one reference task;
- prefer a small explicit primitive over a broad human-oriented abstraction;
- prefer observable semantics over implicit magic.

When choosing whether to defer something:
- defer it if it is mainly UX polish for CLI/HTTP;
- defer it if it depends on product-specific workflow semantics;
- defer it if it broadens the shell without improving the core software-work loop.

## Immediate Next Actions

The next kernel-focused loop should start with:
- hardening default filesystem boundary enforcement;
- defining the virtual `cwd` and path-resolution model after that boundary hardening is in place;
- tightening trace fidelity for mutation-heavy operations;
- reviewing cancellation/context handling across filesystem and execution paths;
- converting current review findings into explicit work items with validation commands.

## Supporting Material

External research and field notes collected for this plan:
- `task_outputs/research/agent-runtime-papers-summary-2026-03-22.md`
- `task_outputs/research/frontier-agent-team-blog-podcast-notes-2026-03-22.md`
- `knowledge_base/papers/agent-runtime/`

Current execution artifacts:
- `.bagakit/ft-harness/feats/f-20260321-default-filesystem-boundary-enforcement/`
- `.bagakit/ft-harness/feats/f-20260321-virtual-cwd-path-resolution/`
- `.bagakit/ft-harness/feats/f-20260321-mutation-trace-fidelity/`
- `.bagakit/ft-harness/feats/f-20260321-cancel-timeout-effectiveness/`

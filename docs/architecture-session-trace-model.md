---
title: Session and Execution Trace Model
required: false
sop:
  - Read this doc before adding session state, structured execution results, or policy override semantics.
  - Keep session and trace contracts generic; product semantics belong in adapters.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Session and Execution Trace Model

## Context
`simsh` currently executes commands as mostly one-shot invocations. That keeps the runtime simple, but it leaves three gaps for agent harnesses:

- no first-class session primitive for cross-call state and resume;
- no structured execution result contract for machine consumption;
- no execution trace contract that records read/write/deny side effects in a form planners can use directly.

This document defines the next core contract layer without coupling the runtime to any single business domain.

## Goals
- Introduce a generic `Session` primitive for cross-call execution state.
- Replace text-only command results with a structured `ExecutionResult` contract.
- Add a machine-consumable `ExecutionTrace` contract for side effects and policy outcomes.
- Support session-scoped policy with predictable per-execution narrowing and explicit elevation rules.

## Non-Goals
- Encoding business meaning into session boundaries.
- Solving multi-agent tenancy or workspace orchestration in the first session model.
- Defining domain memory schemas or product-specific trace consumers here.

## Core Decision
A session is a runtime execution context, not a product object.

The embedding platform decides whether a session maps to one user flow, one background task, one long-running tool interaction, or something else. `simsh` only guarantees the runtime semantics of that context.

## Session Contract

### Required Properties
- `session_id`: stable identifier for resume/checkpoint/close flows.
- `created_at`, `updated_at`: lifecycle timestamps.
- `profile`: baseline compatibility profile for the session.
- `policy_ceiling`: maximum allowed execution policy for the session.
- `state`: minimal runtime state needed across calls.

### Session State Scope
Session state SHOULD cover only runtime concerns such as:
- command aliases and environment seeded by rc/bootstrap logic;
- runtime-level session metadata needed for resume/checkpoint;
- adapter-scoped opaque continuation metadata.

Session state SHOULD NOT embed product objects directly.

### Lifecycle
- `create`: initialize a session with baseline profile/policy and adapter hooks.
- `resume`: reopen a prior session using persisted checkpoint state.
- `checkpoint`: persist resumable runtime state plus adapter continuation data.
- `close`: flush terminal state, emit final audit/trace hooks, and release resources.

Requests without a session identifier MAY run as ephemeral one-shot sessions.

## Policy Scope
- A session owns a `policy_ceiling`.
- Each execution may request the same or a narrower policy.
- Widening above the session ceiling requires an explicit control-plane action and MUST be auditable.
- Profiles SHOULD be treated as session baseline configuration unless a narrower capability mode is explicitly supported later.

## ExecutionResult Contract

### Required Fields
- `execution_id`
- `session_id` when applicable
- `exit_code`
- `stdout`
- `stderr`
- `started_at`
- `finished_at`
- `duration_ms`
- `trace`

### Notes
- CLI rendering may stay human-friendly by default, but HTTP and embedding APIs SHOULD expose the structured form directly.
- Backward-compatible text shims MAY flatten `stdout` for legacy surfaces, but the structured result becomes the source of truth.

Example shape:

```json
{
  "execution_id": "exec_123",
  "session_id": "sess_456",
  "exit_code": 0,
  "stdout": "ok\n",
  "stderr": "",
  "started_at": "2026-03-11T09:00:00Z",
  "finished_at": "2026-03-11T09:00:00Z",
  "duration_ms": 124,
  "trace": {}
}
```

## ExecutionTrace Contract

### Purpose
`ExecutionTrace` is for machine consumption by planners, adapters, and policy layers. It is not just an audit log dump.

### Required Categories
- command identity: command name, argv, pipeline structure when applicable;
- policy/profile used: effective policy, effective profile, truncation or downgrade markers;
- timing: duration and timeout/cancel markers;
- path effects:
  - `requested_paths`
  - `read_paths`
  - `written_paths`
  - `appended_paths`
  - `edited_paths`
  - `created_dirs`
  - `removed_paths`
  - `denied_paths`
- resource summary: bytes read, bytes written, output truncated flags.

### Design Rules
- Traces SHOULD distinguish attempted operations from successful operations.
- Traces SHOULD record direct side effects, not inferred heuristics.
- Builtins and external commands should project into the same high-level trace categories even if collection mechanisms differ.

## Relationship to Audit
- `AuditSink` remains an observability hook for out-of-band pipelines.
- `ExecutionTrace` is returned to the caller as part of `ExecutionResult`.
- The two may share fields, but they serve different consumers and should not be conflated.

## Relationship to Path Metadata
- Path access semantics remain defined by `PathMeta.access` and `PathMeta.capabilities`.
- `ExecutionTrace` records what happened during execution.
- `/v1/execute` metadata and trace output SHOULD align on path identity and access semantics.

## Snapshot and Resume
- Session checkpoints SHOULD capture the minimum state needed for deterministic resume.
- Adapter-managed state belongs behind explicit adapter hooks, not inside generic core structs.
- Closing a session SHOULD provide a hook point for adapter-side flushes such as memory snapshots or reindex triggers.

## First Implementation Boundary
The first implementation SHOULD land:
- core session identifiers and lifecycle hooks;
- structured `ExecutionResult`;
- trace categories for builtin file operations and external command execution;
- session-scoped policy narrowing rules.

It SHOULD defer:
- workspace/multi-agent tenancy;
- domain-specific trace consumers;
- business-specific session taxonomy.

## Validation Matrix
- One-shot execution still works without explicit session setup.
- Repeated requests within a session preserve runtime state intentionally, not accidentally.
- Per-request policy narrowing works; privilege widening is rejected unless explicitly authorized.
- Read/write/deny path effects are visible in structured results.
- At least one adapter-backed workload validates that the returned trace is actionable, not just log noise.

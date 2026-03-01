# Feat Proposal: f-20260301-structured-execution-result-contract

## Why
- Core command execution still collapses results to text plus exit code, which makes machine consumption lossy and pushes parsing burden to adapters and HTTP clients.
- The structured result contract needs to become the source of truth before trace and adapter lifecycle work can compose cleanly on top.

## Goal
- Introduce structured ExecutionResult contracts across engine, runtime, and HTTP surfaces while preserving CLI compatibility.

## Scope
- In scope:
- new `ExecutionResult` contract shape in core runtime and transport layers
- CLI flattening rules for human-facing output compatibility
- HTTP response changes that expose structured stdout/stderr/result metadata directly
- Out of scope:
- detailed trace population for read/write/deny sets
- adapter lifecycle hooks and memory protocol
- multi-agent or workspace semantics

## Impact
- Code paths:
- `pkg/engine`
- `pkg/engine/runtime`
- `pkg/contract`
- `pkg/service/httpapi`
- `cmd/simsh-cli`
- Tests:
- engine/runtime contract tests
- HTTP response shape tests
- CLI compatibility tests
- Rollout notes:
- preserve old text rendering via adapters instead of keeping text-only core APIs as SSOT

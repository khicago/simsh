---
title: Daily learnings (2026-03-23)
kind: howto
confidence: high
tags:
  - testing
  - coverage
  - kernel
sources:
  - pkg/contract/core_contract_test.go
  - pkg/sh/runtime_env_test.go
  - pkg/fs/filesystem_env_additional_test.go
  - pkg/engine/runtime/runtime_core_additional_test.go
  - pkg/mount/mount_core_test.go
  - pkg/adapter/localfs/adapter_behavior_test.go
created: 2026-03-23T00:00:00Z
updated: 2026-03-23T00:00:00Z
---

## Context

This session focused on raising `simsh` core test coverage without reward hacking. The goal was to improve confidence in kernel contracts, not to maximize percentages by testing low-value wrappers.

## Learnings

- For `simsh`, the highest-ROI coverage work is in `pkg/contract`, `pkg/fs`, `pkg/engine/runtime`, mount behavior, and default filesystem semantics. These define trust boundaries and runtime contract shape.
- Thin wrappers should only be tested when they are the public seam. `pkg/sh` was worth a smoke test because it is the exported shell-runtime facade; deeper behavior still belongs to `engine` and `builtin` tests.
- Good coverage work in this repo should pair a normal-path test with at least one edge/failure-path test for the same semantic area:
  - policy ceiling / write-limit helpers
  - path normalization and command-reference errors
  - nil-runtime and default-factory behavior
  - blank-host-root fallback for filesystem setup
  - mount-backed missing/unsupported paths
- When a test fails, first verify whether the assumption is wrong before touching production code. Two examples from this session:
  - `NormalizeEnvVars` correctly accepts `PATH`
  - `runtimeOptionsFromSession` correctly clears `RCFiles` to avoid replaying RC bootstrap

## Scope

Apply this approach when improving coverage for kernel/runtime packages. Do not use it as justification to add low-signal tests for generated constants, trivial accessors, or renderer branches with no contract risk.

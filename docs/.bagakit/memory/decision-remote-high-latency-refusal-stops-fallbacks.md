---
title: remote_high_latency refusal must stop builtin and engine fallbacks
kind: decision
tags:
  - decision
  - mount
  - performance
  - release-gate
sources:
  - docs/architecture-high-performance-mount-system.md
  - docs/notes-v0-2-x-to-v0-3-0-migration.md
  - pkg/contract/mount_dispatch.go
  - pkg/contract/mount_unsupported.go
  - pkg/engine/virtualfs_bridge.go
  - pkg/builtin/op_listing.go
  - pkg/builtin/op_json.go
  - pkg/builtin/search_scan.go
created: 2026-04-06T09:05:00Z
confidence: high
updated: 2026-04-06T09:05:00Z
---

## Context

- `v0.3.0` now treats high-performance mount behavior as a release-gate concern.
- The runtime already had explicit refusal logic for `remote_high_latency` mounts, but some builtin helpers still treated those refusals as ordinary `ErrUnsupported` and silently fell back into local fanout loops.

## Decision

- `remote_high_latency` capability refusal must remain distinguishable from ordinary unsupported fallback.
- Builtin and engine helpers may still fall back for `local_fast` or other explicitly allowed unsupported cases, but they must not do so for `remote_high_latency` capability gaps.
- The shared mechanism is `MountUnsupportedError` plus `AllowsUnsupportedFallback`.

## Why

- The architectural contract is not just “unsupported”; it is “fail closed or narrow scope” for high-latency mounts.
- Without a shared error classification, helper layers drift toward optimistic fallback even when contract code already refuses.
- A small shared mechanism is lower risk than widening the mount API or introducing speculative caching just to avoid proving the current contract.

## Scope

- Applies to mount-backed list, enumerate, bulk-read, content-search, and mutation-batch flows.
- Does not remove legitimate local fallback behavior for `local_fast` or other explicitly allowed mount classes.

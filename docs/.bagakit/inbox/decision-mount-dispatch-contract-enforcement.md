---
title: Mount dispatch now enforces CLI class, ordering, and consistency contracts
kind: decision
status: inbox
tags:
  - decision
  - mount
  - dispatch
  - capability-contract
sources:
  - docs/architecture-high-performance-mount-system.md
  - pkg/contract/mount_dispatch.go
  - pkg/engine/virtualfs_bridge.go
  - pkg/contract/path_ops.go
  - pkg/builtin/op_copy.go
  - pkg/builtin/op_move.go
created: 2026-04-05T09:45:54Z
---

## Context
The mount-system refactor introduced semantic axes, `SupportedCLIClasses`, and consistency contracts in docs and code, but the first implementation pass still had gaps:

- `ReadMany` could reorder results across mount groups.
- `SearchContent` dispatch depended on the first target instead of the full target set.
- `SupportedCLIClasses` and `Consistency` existed in profile metadata but were not enforced as runtime gates.
- `cp` / `mv` reused generic `PathOpRead`, which blurred ordinary readable paths with transfer-source semantics.

## Decision
- Mount dispatch now treats `SupportedCLIClasses` as a real execution contract, not decorative metadata.
- Multi-file mount reads must preserve the caller's requested path order even when requests span filesystem and multiple mounts.
- Search dispatch must evaluate the full target set before choosing mount-vs-filesystem routing; mixed filesystem/mount search targets fail closed.
- Writable mounts must declare either visibility guarantees or explicit `refresh_required` semantics before mutation dispatch is allowed.
- Copy/move source preflight now uses a dedicated `PathOpTransferSource` instead of overloading generic read.

## Rationale
- Agent flows depend on deterministic ordering for `cat a b c`, batch `json`, and batch `frontmatter`; map-iteration ordering is unacceptable.
- Target-order-dependent dispatch hides capability errors and makes behavior depend on arbitrary argument ordering.
- If `SupportedCLIClasses` are not enforced, the profile stops being SSOT and the architecture doc becomes aspirational only.
- Visibility contracts matter for writable mounts because read-after-write/list-after-write/search-after-write semantics directly affect agent planning and verification loops.
- Transfer-source semantics are more specific than ordinary readability, so they should not ride on a generic `read` preflight bit.

## Scope
- Applies to mount dispatch, mount-aware engine routing, and command preflight behavior.
- Especially relevant for `ls`, `find`, `grep`, `rg`, `cat`, `cp`, `mv`, and any future writable factual mount work.
- Does not require projection mounts to become writable; it tightens the routing contract for mounts that do opt into more CLI classes.

## Promote To
- `docs/.bagakit/memory/decision-mount-dispatch-contract-enforcement.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

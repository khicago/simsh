# Feat Proposal: f-20260217-vfs-mount-ancestors

## Why
- simsh exposes commands via virtual mounts (e.g. `/sys/bin`, `/bin`). Agents frequently navigate mount trees as part of discovery and execution.
- Without exposing mount parent directories, mount points feel "disconnected" (e.g. `/sys/bin` exists but `ls /sys` fails), which increases agent confusion and brittle prompt logic.
- Mount-backed trees must be immutable to keep execution deterministic and safe:
  - prevent accidental modification of builtin/external command mounts
  - avoid partial side effects for multi-step mutations (e.g. `mv` writing destination then failing on source removal)

## Goal
- 允许显式访问 mount 的父目录（如 /sys），并对 mount/synthetic 路径统一施加不可变写保护（rm/mv/cp/mkdir/重定向等），避免 mv 半写入。

## Scope
### In scope (implemented)
- Expose **synthetic parent directories** for active mount points (e.g. `/sys` for `/sys/bin`) so `ls /sys` works and shows mount children.
- Enforce **immutability** for:
  - mount-backed paths (e.g. `/sys/bin/ls`)
  - synthetic mount-parent paths (e.g. `/sys`)
  across `write/edit/remove/mkdir/cp/mv` flows.
- Add a **preflight hook** for multi-step commands to check access early and avoid partial writes.

### Out of scope (this feat)
- Full ACL/permission system beyond `ro/rw` semantics.
- Making mounts writable.
- Backwards compatibility for `ls -l` output formatting (to be defined in follow-up design).

## Impact
### Code paths (implemented)
- `pkg/engine/virtualfs_bridge.go`: mount router + synthetic directory exposure + immutable guards.
- `pkg/contract/integration_contract.go`: `Ops` extended with `CheckPathOp` hook.
- `pkg/contract/path_ops.go`: `PathOp` enum for preflight checks.
- `pkg/builtin/op_copy.go`, `pkg/builtin/op_move.go`, `pkg/builtin/op_mkdir.go`, `pkg/builtin/op_remove.go`: call `CheckPathOp` to avoid partial writes.
- `pkg/sh/runtime_env.go`, `docs/architecture.md`, `README.md`, `simsh.md`: user-facing semantics notes.

### Tests (implemented)
- `pkg/engine/engine_test.go`:
  - `TestEngineBuiltinAndExternalMounts` includes synthetic `/sys` listing
  - `TestEngineMountParentIsImmutable` exercises write/mkdir/rm/cp/mv blocks for mount and synthetic parent

### Rollout notes
- No host filesystem permissions are changed; immutability is enforced in the virtual overlay layer.
- Mount parent exposure is synthetic and only exists when descendant mounts are active.

## Design Addendum (planned): Path Access SSOT + Listing/API Formats

This section captures follow-up design so agents can *understand* access constraints without trial-and-error, and so CLI/man/API stay consistent.

### Motivation
- `ls -l` output is currently verbose and not optimized for agent token budgets (especially for large directories).
- Both humans and agents benefit from an explicit `ro/rw` status and machine-readable capabilities.

### SSOT: `access` + `capabilities`
- Extend `contract.PathMeta` (and `LSLongRow`) with:
  - `access`: `"ro" | "rw"`
  - `capabilities`: string list describing allowed operations (see doc for canonical set)
- **SSOT rule**: access/capabilities are computed once (DescribePath / overlay wrappers) and consumed by:
  - `ls -l` (short columns by default)
  - `ls -l --fmt md|json`
  - `/v1/execute` API (opt-in `include_meta=true`)
  - manuals (`man ls`) and injected runtime docs (`simsh.md`) via a single schema definition.

### `ls -l` output formats
- Default long listing uses short columns (no repeated `key=value`), plus a one-line legend at the end.
- Add `--fmt md` and `--fmt json` for high-signal human tables and machine parsing.

### API meta
- Extend `/v1/execute` request with `include_meta` (default false).
- When enabled, response includes path access metadata (access + capabilities + denials) derived from SSOT.

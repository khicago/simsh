# Feat Proposal: f-20260218-path-access-metadata

## Why
- Agents need a deterministic way to understand whether a path is writable vs immutable without trial-and-error.
- `ls -l` is the primary discovery surface, but its current long format is verbose and not token-efficient for directories with hundreds of entries.
- We already enforce mount immutability and expose synthetic mount-parent directories; this feat makes those rules explicit and consumable via a single SSOT schema.

## Goal
- Implement SSOT path access metadata (access=ro|rw + capabilities[]) and surface it consistently in ls -l (short columns + legend, --fmt md/json) and /v1/execute (include_meta opt-in).

## Scope
### In scope
- Extend `contract.PathMeta` + `contract.LSLongRow` with:
  - `access: ro|rw`
  - `capabilities[]` (compact allowed-ops set)
- Compute access/capabilities once (DescribePath / overlay wrappers) and consume it in:
  - `ls -l` (default compact columns + legend)
  - `ls -l --fmt md|json` (optional alternate formats)
  - HTTP `/v1/execute` response metadata (opt-in `include_meta=true`)
- Update `man ls` / embedded manuals to document the new columns/flags.
- Add tests covering mount-backed + synthetic mount-parent access metadata and ls formatting.

### Out of scope
- Full ACL model (user/group/mode bits).
- Making mount-backed trees writable.
- Precise per-operation provenance tracking for `/v1/execute` meta beyond the initial schema.

## Impact
### Code paths
- `pkg/contract/runtime_types.go`: schema extension (`PathMeta`, `LSLongRow`)
- `pkg/engine/virtualfs_bridge.go`: mount/synthetic DescribePath access semantics
- `pkg/adapter/agentfs`, `pkg/adapter/localfs`: zone/local root access semantics
- `pkg/builtin/op_listing.go`: `ls -l` output formatting + `--fmt` support
- `pkg/service/httpapi/execute_endpoint.go`: optional meta on `/v1/execute`

### Tests
- `pkg/engine/engine_test.go`: listing output includes access column for mount-backed + synthetic paths
- `pkg/service/httpapi/handler_test.go`: (later task) include_meta response shape

### Rollout notes
- Keep default output compact; alternate formats are opt-in.
- Treat `docs/architecture-path-access-metadata.md` as the normative schema reference and update it when behavior changes.

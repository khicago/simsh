---
title: Memory/Skills Extension Architecture
required: false
sop:
  - Read this doc before adding memory/resource/skill mounts or context-framework features.
  - Keep core-runtime boundaries explicit: core exposes extension contracts, business layers own retrieval/index/evolution behavior.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Memory/Skills Extension Architecture

## Decision

simsh core stays focused on deterministic execution and virtual filesystem routing. We do **not** implement a business opinionated memory/skills/evolution system inside core packages.

Instead, core provides a stable extension boundary, and business layers build domain features on top.

## Core Extension Surface (already available)

Use these as the only required integration surface:

- `contract.VirtualMount`: pluggable subtree mounts (`pkg/contract/mount_contract.go`).
- `contract.Ops` callbacks and `PathMeta`: SSOT semantics for `access` + `capabilities` (`pkg/contract/integration_contract.go`, `pkg/contract/runtime_types.go`).
- mount router merge/dispatch and synthetic parent directories (`pkg/engine/virtualfs_bridge.go`).

Core guarantees that mount-backed paths and synthetic mount-parent paths are immutable for write/mkdir/remove/copy/move flows.

## Recommended Business-Layer Framework

### 1) Driver layer (filesystem adapters)

Implement dedicated drivers that project business data into read-only mount trees:

- memory driver: e.g. `/memory`
- resources driver: e.g. `/resources`
- skills driver: e.g. `/skills`

Each driver implements `VirtualMount` and only exposes deterministic read/list/search/describe behavior.

### 2) Service layer (domain logic)

Implement domain-specific capabilities outside core:

- memory indexing + semantic retrieval
- skill discovery + eligibility filtering + precedence
- evolution lifecycle (versioning, deprecation, migration, curation)

This layer may use DB/vector/search sidecars, but should expose a filesystem-friendly projection via driver layer.

### 3) Control plane (write path)

Keep write/update flows explicit and auditable via dedicated commands/APIs (not implicit file writes to mounts):

- register/unregister/update skill entries
- promote/curate memory entries
- trigger reindex/snapshot/hot-reload

## Suggested Conventions

- Default mounts to read-only; use `PathMeta.access/capabilities` for planner-facing clarity.
- Define source precedence clearly (recommended: workspace > user-managed > bundled).
- Add load-time gating for skills (required bins/env/config/os), but keep fallback predictable.
- Handle missing memory/resource files gracefully (`not found` should be a normal result path, not a hard failure).
- Apply limits from day one (scan count, loaded entries, prompt injection size, debounce/watch intervals).

## OpenClaw-Inspired Practices (reference)

The following practices are good references for business-layer design (do not copy them into core):

- skills source precedence + per-skill gating and config overlays:
  - `/Users/bytedance/proj/priv/bagaking/openclaw/docs/tools/skills.md`
  - `/Users/bytedance/proj/priv/bagaking/openclaw/docs/tools/skills-config.md`
  - `/Users/bytedance/proj/priv/bagaking/openclaw/src/config/types.skills.ts`
- memory as Markdown source-of-truth + plugin-slot backend switching:
  - `/Users/bytedance/proj/priv/bagaking/openclaw/docs/concepts/memory.md`
  - `/Users/bytedance/proj/priv/bagaking/openclaw/src/config/types.plugins.ts`
- event-driven evolution hooks (session-memory flush, bootstrap augmentation, command audit):
  - `/Users/bytedance/proj/priv/bagaking/openclaw/docs/automation/hooks.md`

## Rollout Plan

1. Stage A: define mount namespaces + minimal read-only drivers.
2. Stage B: add service-layer indexing/gating and stable metadata schema.
3. Stage C: add control-plane APIs/commands for evolution workflows.
4. Stage D: add observability and policy checks (latency, cache hit, denials, audit).

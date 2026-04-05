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

This is the intended layering for harnesses and AgentOS-style systems:
- `simsh` owns execution and virtual path semantics
- adapters own memory and skill projection
- higher-level systems own retrieval, curation, planning, and orchestration

## Core Extension Surface (already available)

Use these as the only required integration surface:

- `contract.VirtualMount`: pluggable subtree mounts (`pkg/contract/mount_contract.go`).
- `contract.Ops` callbacks and `PathMeta`: SSOT semantics for `access` + `capabilities` (`pkg/contract/integration_contract.go`, `pkg/contract/runtime_types.go`).
- mount router merge/dispatch and synthetic parent directories (`pkg/engine/virtualfs_bridge.go`).

Core guarantees that mount-backed paths and synthetic mount-parent paths are immutable for write/mkdir/remove/copy/move flows.

For the broader performance and capability model of non-trivial mounts, especially when the backend is RPC-, DB-, or search-backed, see `docs/architecture-high-performance-mount-system.md`.

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

This is the layer where an AgentOS or harness should decide:
- what counts as memory
- how memory is ranked or surfaced
- when new observations become durable memory
- how skills are selected or gated for a given run

This layer may use DB/vector/search sidecars, but should expose a filesystem-friendly projection via driver layer.

### 3) Control plane (write path)

Keep write/update flows explicit and auditable via dedicated commands/APIs (not implicit file writes to mounts):

- register/unregister/update skill entries
- promote/curate memory entries
- trigger reindex/snapshot/hot-reload
- refresh or invalidate projected resources/documents
- advance adapter-local workflow or curation state when the adapter owns that policy

## Suggested Conventions

- Default mounts to read-only; use `PathMeta.access/capabilities` for planner-facing clarity.
- Define source precedence clearly (recommended: workspace > user-managed > bundled).
- Keep projection freshness explicit and small-state, not prose-only. Prefer canonical states such as `snapshot`, `live`, `stale`, and `updated`, and surface them in sidecars or managed summary views.
- Keep refresh ownership in the control plane. If a projection becomes stale, expose that staleness in `/memory` or sidecar metadata, but do not make the projected mount itself the write path for refresh.
- Treat `/memory` as a managed read-only view over adapter state. Raw observations, projection indexes, curated summaries, and workflow views may share the namespace, but they should remain conceptually distinct and auditable.
- A minimal current reference shape for curation is: explicit control-plane promotion into `/memory/curated.json`, an optional human-readable mirror at `/memory/curated.md`, and stable source-path provenance on each curated entry.
- For `/skills`, prefer a stable read-only projection plus explicit metadata over implicit gating. The current reference shape is: projected skill artifacts under `/skills/...`, an index at `/skills/_index.json`, and mirrored metadata in `/memory/projections.json` that surfaces `source`, `freshness`, `eligibility`, `precedence`, and optional selection state.
- If an adapter exposes skill selection, keep the competition boundary explicit. Selection should be derived inside an adapter-defined scope from eligibility plus precedence, not inferred from mount writability and not guessed from path layout alone.
- When non-selected skills remain visible, surface why they lost. The current reference shape is a `selection` object with `state`, `mode`, `scope`, `reason`, and optional `winner_path`, alongside a compatibility `selected` bit for quick consumers.
- Unscoped skills stay in compatibility mode: they may surface explicit selected or not-selected state, but the adapter must not invent competition from path layout alone.
- A minimal current reference control-plane shape for skills is: explicit adapter-local `upsert/update/remove` APIs, read-only `/skills` projection refresh on the next projection rebuild, and reselection through the same SSOT derivation path rather than ad hoc mutation rules.
- A minimal current reference audit shape for skill control-plane mutations is: `/memory/skills_audit.json` for ordered machine-readable events, `/memory/skills_audit.md` for compact human review, and explicit projection-generation visibility timing instead of implicit diff-based inference.
- A minimal current reference metrics and denial shape is: `/memory/projection_metrics.json` plus `/memory/projection_metrics.md` for compact counts and generation data, and `/memory/denials.json` plus `/memory/denials.md` for unique denied-path samples and namespace buckets. Cache-hit metrics stay explicitly unavailable until a real cache exists.
- Add load-time gating for skills (required bins/env/config/os), but keep fallback predictable.
- Handle missing memory/resource files gracefully (`not found` should be a normal result path, not a hard failure).
- Apply limits from day one (scan count, loaded entries, prompt injection size, debounce/watch intervals).

## Rollout Plan

1. Stage A: define mount namespaces + minimal read-only drivers.
2. Stage B: add service-layer indexing/gating and stable metadata schema.
3. Stage C: add control-plane APIs/commands for evolution workflows.
4. Stage D: add observability and policy checks (latency, cache hit, denials, audit).

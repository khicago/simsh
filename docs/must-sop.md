# Project SOP

This SOP is generated from docs frontmatter. Do not edit manually.

## Update Requirements
- When a document with SOP frontmatter changes, regenerate this file and commit the result:
  - `export BAGAKIT_LIVING_DOCS_SKILL_DIR="${BAGAKIT_LIVING_DOCS_SKILL_DIR:-${BAGAKIT_HOME:-$HOME/.bagakit}/skills/bagakit-living-docs}"`
  - `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/living-docs-generate-sop.sh" .`
- Add new SOP items by updating the `sop` list in the source document frontmatter.
- Keep SOP items small and actionable; use the source document for details.

## SOP Items

### High-Performance Mount System
Source: `docs/architecture-high-performance-mount-system.md`
- Read this doc before changing mount abstractions, adapter-backed filesystem projection behavior, or tool flows that can amplify mount access patterns.
- Keep mount design explicit along semantic axes, capability contracts, and latency/consistency guarantees instead of relying on ad hoc fallback behavior.
- When tool or mount changes would otherwise fall back to per-file RPC fanout, require explicit refusal or scope narrowing for `remote_high_latency` mounts instead of documenting the fallback and moving on.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Memory/Skills Extension Architecture
Source: `docs/architecture-memory-skills-extension.md`
- Read this doc before adding memory/resource/skill mounts or context-framework features.
- Keep core-runtime boundaries explicit: core exposes extension contracts, business layers own retrieval/index/evolution behavior.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Architecture Overview
Source: `docs/architecture-overview.md`
- Read this doc before changing the high-level narrative order of simsh architecture.
- Update this doc when the relationship between kernel, default workspace, adapters, and entry surfaces changes.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Paired A/B Uplift Proof Harness
Source: `docs/architecture-paired-ab-uplift-proof-harness.md`
- Read this doc before changing the paired uplift benchmark, baseline substrate, or per-run report schema.
- Keep the experiment controlled: hold the agent, paired task set, and budgets fixed while changing only the runtime substrate.
- Keep the paired uplift layer downstream from the native benchmark suite and external comparison artifacts; do not mutate those layers to make uplift look better.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Path Access Metadata and Listing/API Formats
Source: `docs/architecture-path-access-metadata.md`
- Read this doc before changing PathMeta access/capabilities, ls -l output formats, or /v1/execute metadata.
- Update this doc when schema/flags/output formats change.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Platform Adapter Contract
Source: `docs/architecture-platform-adapter-contract.md`
- Read this doc before defining adapter lifecycles, RPC-to-file projections, or memory lifecycle protocols.
- Keep adapter contracts explicit at the platform boundary; do not push product semantics into core by default.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Session and Execution Trace Model
Source: `docs/architecture-session-trace-model.md`
- Read this doc before adding session state, structured execution results, or policy override semantics.
- Keep session and trace contracts generic; product semantics belong in adapters.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Doc Co-Authoring Workflow (Guidelines)
Source: `docs/guidelines-doc-coauthoring.md`
- Use this workflow when drafting substantial docs (proposals, specs, decision docs, RFCs).
- Update this doc when your team changes its doc-writing workflow or quality bar.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Maintaining Reusable Items (可复用项维护)
Source: `docs/norms-maintaining-reusable-items.md`
- At the start of each iteration, check whether the project needs a new reusable-items catalog for an active domain (coding/design/writing/knowledge) and create/update it.
- When introducing or updating a reusable item (component/library/mechanism/token/style pattern/index; including API/behavior/ownership/deprecation), verify the relevant catalog entry is correct and update it in the same change.
- When SOP/frontmatter changes in these docs, regenerate `docs/must-sop.md` with `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_generate_sop.sh" .`.

### Adopting Living Docs in an Existing Project
Source: `docs/notes-adopting-living-docs.md`
- Read this doc before adopting Bagakit living-docs into a repo that already has documentation under `docs/`.
- Update this doc when the repo's adoption/migration strategy changes (e.g., naming rules, CI gating, or legacy-doc handling).
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Builtin ACI Review
Source: `docs/notes-builtin-aci-review.md`
- Read this doc when reviewing builtin command UX, manuals, or output contracts for agent use.
- Update this doc when builtin default formats, command metadata, or manual-summary strategy changes.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Continuous Learning (Default)
Source: `docs/notes-continuous-learning.md`
- At the end of a Bagakit Agent work session, capture a draft learning note into `docs/.bagakit/inbox/` (manual or via `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_learning.sh" extract --root . --last`). The default extractor upserts into a daily file to avoid fragmentation.
- Weekly (or before major releases), review `docs/.bagakit/inbox/` and promote durable items into `docs/.bagakit/memory/`.
- When promoting, keep entries short and source-linked; prefer `decision-*`/`preference-*`/`gotcha-*`/`howto-*` over long narratives. If the curated target already exists, merge instead of creating duplicates.

### Response Directives (Examples)
Source: `docs/notes-directives-examples.md`
- Read this doc when you want to introduce or change response directives (`directives:` in doc frontmatter).
- Keep directive usage practical: only add directives that reduce mistakes or improve debuggability.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### First Version Plan (Historical Retrospective)
Source: `docs/notes-first-version-plan.md`
- Read this doc only when you need historical context about the first implementation wave and completed hardening/tooling work.
- Do not use this doc as the current kernel roadmap; use `docs/notes-kernel-optimization-plan.md` and `docs/notes-kernel-execution-backlog.md` instead.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Kernel Execution Backlog
Source: `docs/notes-kernel-execution-backlog.md`
- Read this doc when choosing the next kernel execution item or converting review findings into implementation work.
- Update this doc when kernel execution items are added, reprioritized, completed, or superseded.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Kernel Optimization Plan
Source: `docs/notes-kernel-optimization-plan.md`
- Read this doc when planning kernel optimization work or reviewing runtime tradeoffs.
- Update this doc when kernel priorities, sequencing, or validation gates change.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Project Charter
Source: `docs/notes-project-charter.md`
- Read this doc before major architecture or product-boundary changes.
- Update this doc when project goals, scope, or non-goals change.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Requirements Baseline
Source: `docs/notes-requirements.md`
- Read this doc before changing current kernel-facing product requirements or implementation priorities.
- Update this doc when new cross-cutting requirements become the source of truth for ongoing workstreams.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Reusable Items - Coding (Catalog)
Source: `docs/notes-reusable-items-coding.md`
- Update this list when you introduce or adopt a new reusable component/library/mechanism.
- When you remove or deprecate something, update this list and point to the replacement or migration.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Migration Plan: v0.1.0 to v0.2
Source: `docs/notes-v0-1-0-to-v0-2-migration.md`
- Read this doc before upgrading integrations from the v0.1.0 release baseline to the released v0.2 contract set.
- Update this doc when the v0.2 feat order, scope, or compatibility strategy changes.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Migration Guide: v0.2.x to v0.3.0
Source: `docs/notes-v0-2-x-to-v0-3-0-migration.md`
- Read this doc before upgrading integrations from the v0.2.x release line to the released v0.3.0 line.
- Update this doc when the v0.3.0 migration guidance or compatibility notes change.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### v0.3.0 Release Readiness
Source: `docs/notes-v0-3-0-release-readiness.md`
- Read this doc when you need the historical closeout artifact for the v0.3.0 line.
- Update this doc only if the recorded v0.3.0 closeout history or references need correction.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### v0.3.1 Patch Release Readiness
Source: `docs/notes-v0-3-1-patch-release-readiness.md`
- Read this doc before cutting the v0.3.1 patch line or claiming the current post-v0.3.0 tree is ready to tag.
- Update this doc when the v0.3.1 patch scope, evidence set, or cut criteria change.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Execute Preflight Performance References
Source: `docs/refs/notes-execute-preflight-performance-refs.md`
- Use this list before major runtime-performance refactors around per-exec setup overhead.
- Update the actionable-takeaways section when adopting a new optimization strategy.
- Regenerate docs/must-sop.md after SOP/frontmatter changes.


# Project SOP

This SOP is generated from docs frontmatter. Do not edit manually.

## Update Requirements
- When a document with SOP frontmatter changes, regenerate this file and commit the result:
  - `export BAGAKIT_LIVING_DOCS_SKILL_DIR="<path-to-bagakit-living-docs-skill>"`
  - `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/living-docs-generate-sop.sh" .`
- Add new SOP items by updating the `sop` list in the source document frontmatter.
- Keep SOP items small and actionable; use the source document for details.

## SOP Items

### Memory/Skills Extension Architecture
Source: `docs/architecture-memory-skills-extension.md`
- Read this doc before adding memory/resource/skill mounts or context-framework features.
- Keep core-runtime boundaries explicit: core exposes extension contracts, business layers own retrieval/index/evolution behavior.
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

### Project Charter
Source: `docs/notes-project-charter.md`
- Read this doc before major architecture or product-boundary changes.
- Update this doc when project goals, scope, or non-goals change.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Reusable Items - Coding (Catalog)
Source: `docs/notes-reusable-items-coding.md`
- Update this list when you introduce or adopt a new reusable component/library/mechanism.
- When you remove or deprecate something, update this list and point to the replacement or migration.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Migration Plan: v0.1.0 to v0.2
Source: `docs/notes-v0-1-0-to-v0-2-migration.md`
- Read this doc before upgrading integrations from the v0.1.0 release baseline to the planned v0.2 contract set.
- Update this doc when the v0.2 feat order, scope, or compatibility strategy changes.
- Regenerate `docs/must-sop.md` after SOP/frontmatter changes.

### Execute Preflight Performance References
Source: `docs/refs/notes-execute-preflight-performance-refs.md`
- Use this list before major runtime-performance refactors around per-exec setup overhead.
- Update the actionable-takeaways section when adopting a new optimization strategy.
- Regenerate docs/must-sop.md after SOP/frontmatter changes.


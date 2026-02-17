---
title: Adopting Living Docs in an Existing Project
required: false
sop:
  - Read this doc before adopting Bagakit living-docs into a repo that already has documentation under `docs/`.
  - Update this doc when the repo's adoption/migration strategy changes (e.g., naming rules, CI gating, or legacy-doc handling).
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Adopting Living Docs in an Existing Project

This doc is a practical adoption playbook for introducing Bagakit living-docs into a repo that already has documentation.

## Goals
- Adopt without breaking existing workflows.
- Make `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/validate-docs.sh" .` pass eventually, without needing a big-bang migration.
- Keep the system maintainable (avoid one-off exceptions unless recorded).

## Step 0: Run Apply
In the repo root:
- Resolve tooling: `export BAGAKIT_LIVING_DOCS_SKILL_DIR="${BAGAKIT_LIVING_DOCS_SKILL_DIR:-${BAGAKIT_HOME:-$HOME/.claude}/skills/bagakit-living-docs}"`
- Apply: `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/apply-living-docs.sh" .`
- Then validate: `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/validate-docs.sh" .` (expect failures on first run in older repos).

## Step 1: Pick an Adoption Mode (Choose One)
### Mode A (Recommended): Incremental Migration
Use this when you want `docs/` to remain the canonical doc root.

1) Keep `docs/` as the doc root.
2) Gradually migrate legacy docs to the Bagakit taxonomy:
   - Rename to type-first: `<type>-<topic>.md` (e.g. `notes-...`, `guidelines-...`).
   - Add the required frontmatter keys: `title`, `required`, `sop`.
3) Only after `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/validate-docs.sh" .` is green, add it to CI.
   - (Command: `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/validate-docs.sh" .`)

### Mode B (Fastest): Move Legacy Docs Out Temporarily
Use this when you have many legacy docs and want to adopt Bagakit immediately.

1) Move legacy docs out of `docs/` temporarily (example: `legacy-docs/`).
2) Keep new/maintained docs under `docs/` using Bagakit conventions.
3) Migrate legacy docs back into `docs/` over time (Mode A rules).

Notes:
- Bagakit validation only checks `docs/` (excluding `docs/.bagakit/`).
- A temporary `legacy-docs/` is allowed, but treat it as an interim state and record the plan.

## Step 2: Be Clear About Managed vs Project-Owned Files
### Managed by Bagakit (may be overwritten on update)
- `AGENTS.md` managed block (between `<!-- BAGAKIT:LIVEDOCS:START -->` and `<!-- BAGAKIT:LIVEDOCS:END -->`).
- `docs/must-*.md` system docs (seeded templates) and `docs/must-sop.md` (generated).
- Tooling lives in the installed skill dir (`$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/`) by default.

Recommendation:
- Do not hand-edit managed areas; update Bagakit templates/scripts and re-apply.
- Optional: if you need a repo-contained toolchain (CI/offline), apply with `--vendor-scripts` and treat the copied scripts as vendor tooling.

### Owned by the project (not overwritten by default)
- Your docs under `docs/`.
- Your memory entries under `docs/.bagakit/{memory,inbox}/`.

## Step 3: Decide on Optional Mechanisms
Bagakit provides optional mechanisms you can adopt (or delete/ignore) based on your project's needs:
- Reusable items governance/catalogs (`docs/norms-maintaining-reusable-items.md`, `docs/notes-reusable-items-*.md`).
- Response directives (`directives:` in doc frontmatter; shown in `docs/must-sop.md`).
- Continuous learning (`docs/notes-continuous-learning.md`).

If you choose not to use an optional mechanism, record that decision (e.g., in memory inbox) to avoid churn.

## Step 4: CI Gating (When Ready)
When the repo is mostly migrated and `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/validate-docs.sh" .` is stable:
- Add a CI step to run:
  - `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/validate-docs.sh" .`
  - `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_generate_sop.sh" .`

---
title: Use living-docs-* scripts when AGENTS references old bagakit_* names
kind: gotcha
status: inbox
tags:
  - gotcha
sources:
  - AGENTS.md
  - /Users/bytedance/proj/priv/bagakit/skills/dist_local/bagakit-living-docs/scripts/living-docs-memory.sh
  - /Users/bytedance/proj/priv/bagakit/skills/dist_local/bagakit-living-docs/scripts/living-docs-inbox.sh
created: 2026-03-24T10:29:51Z
updated: 2026-03-24T10:29:58Z
---

## Candidate
Context:
- During the 2026-03-24 long-run initializer pass, the repo instructions required a Bagakit memory recall search before scoping the next execution item.
- The AGENTS living-docs block still points to `bagakit_memory.sh` and related `bagakit_*` helper names, but the installed skill bundle in this environment exposes `living-docs-memory.sh` and `living-docs-inbox.sh`.

Gotcha:
- If the AGENTS-provided `bagakit_*` script path is missing, use the installed `living-docs-*` script names from the active skill directory instead of assuming the recall tooling is unavailable.
- The recall/search step still works with:
  - `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/living-docs-memory.sh" search '<query>' --root .`
  - `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/living-docs-inbox.sh" new <kind> <topic> --root . --title '<title>'`

Why it matters:
- Skipping recall because of the stale helper name would violate the project living-docs workflow.
- The mismatch is easy to misread as a broken install when it is actually a script rename in the installed skill payload.

Scope:
- Applies when this repo's AGENTS or docs reference `bagakit_*` living-docs helpers but the installed skill only provides `living-docs-*` scripts.
- If the project later updates the managed AGENTS block or installs a skill bundle that restores the old wrapper names, this note can be merged or retired.

## Promote To
- `docs/.bagakit/memory/gotcha-living-docs-script-name-mismatch.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

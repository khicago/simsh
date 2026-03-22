---
title: Continuous Learning (Default)
required: false
sop:
  - At the end of a Bagakit Agent work session, capture a draft learning note into `docs/.bagakit/inbox/` (manual or via `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_learning.sh" extract --root . --last`). The default extractor upserts into a daily file to avoid fragmentation.
  - Weekly (or before major releases), review `docs/.bagakit/inbox/` and promote durable items into `docs/.bagakit/memory/`.
  - When promoting, keep entries short and source-linked; prefer `decision-*`/`preference-*`/`gotcha-*`/`howto-*` over long narratives. If the curated target already exists, merge instead of creating duplicates.
---

# Continuous Learning (Default)

This project uses Bagakit memory (`docs/.bagakit/{inbox,memory}/`) to capture reusable patterns from day-to-day work.

## Repository-Visible System Of Record

For agent work, repository-visible knowledge is the practical system of record.

That means:
- durable instructions should live in `AGENTS.md`, `docs/*.md`, or `docs/.bagakit/memory/*.md`, not only in chat history;
- `AGENTS.md` should stay short and routing-oriented, while durable substance moves into canonical docs;
- inbox items are only a staging area, not a knowledge base.

If a rule, decision, or workflow matters across sessions, it should become repo-visible or it will decay into tribal knowledge.

## Inbox Compression Rule

Do not treat inbox growth as progress.

The desired pattern is:
- capture raw candidates quickly in `docs/.bagakit/inbox/`;
- promote durable items into `docs/.bagakit/memory/`;
- merge repeated placeholders into a smaller number of stronger canonical entries;
- delete task residue once it has been absorbed elsewhere.

The goal is not a large inbox. The goal is a small, high-signal curated memory layer that improves recall quality for later agent work.

## Evolution Direction

As `simsh` moves closer to a harness-engineering stack, continuous learning should evolve in a specific direction:
- less transcript residue, more repository-local system knowledge;
- less monolithic instruction dumping, more explicit canonical docs with narrow scope;
- less one-off feat recap, more synthesis across feat waves;
- more recurring cleanup of stale instructions, stale inbox items, and drift between behavior and docs.

This is not only documentation hygiene. It is part of runtime effectiveness, because agent quality depends on what the repository makes legible.

## Why SOP (no hooks)
Agent runtimes may not provide reliable stop hooks. The SOP above is the default trigger mechanism.

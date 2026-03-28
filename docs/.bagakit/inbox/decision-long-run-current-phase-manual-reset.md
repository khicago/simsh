---
title: Long-run current phase should reset feature-list from manual rows and treat old bagakit-ft rows as historical only
kind: decision
confidence: high
tags:
  - long-run
  - execution-table
  - planning
sources:
  - .bagakit/long-run/bk-execution-table.json
  - .bagakit/long-run/feature-list.json
  - .bagakit/long-run/next-action.json
created: 2026-03-31T15:42:00Z
updated: 2026-03-31T15:42:00Z
---

# Context

The repo had a clean current-phase manual plan in `.bagakit/long-run/bk-execution-table.json`, but `.bagakit/long-run/feature-list.json` and `next-action.json` could still lag behind old execution-table state. Historical `bagakit-ft` rows from earlier feat waves were not supposed to drive the current phase anymore.

# Decision

- Current-phase long-run planning should be driven by `manual-default` rows only until a new upstream execution system is intentionally re-enabled.
- Historical `bagakit-ft` rows remain implementation evidence, but they should not stay in the active feature selector for the current phase.
- When resetting phases, regenerate `feature-list.json` from the current execution table instead of carrying old managed rows forward as tombstones.
- Keep `bk-execution-handoff.md`, `feature-list.json`, and `next-action.json` aligned after the reset so the next initializer/coding pass sees the same current item.

# Rationale

- The current work is an adapter-side realism wave with explicit `IMPLEMENT -> REVIEW -> REFINE` manual rows, not a continuation of the old March feat chain.
- If stale managed rows remain in the feature list, selection quality and doctor output become harder to trust.
- `next-action.json` is only useful when it reflects the same row ordering as the execution table and feature list.

# Scope

- Applies whenever this repo deliberately starts a new current phase from manual rows.
- Does not delete or rewrite archived feat evidence under `.bagakit/ft-harness/feats-archived/`.

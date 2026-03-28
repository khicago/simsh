---
title: Daily learnings (2026-03-24 entry-wrapper tests)
kind: howto
status: inbox
tags:
  - testing
  - tui
  - simshd
  - long-run
sources:
  - pkg/cmd/console_tui_test.go
  - cmd/simshd/main.go
  - cmd/simshd/main_test.go
  - .bagakit/long-run/bk-execution-handoff.md
created: 2026-03-24T05:25:53Z
---

## Candidate

- Context:
  - This session closed the long-run row for deterministic TUI and `simshd` entry-wrapper smoke coverage.
  - The goal was to own the public wrapper behavior without pulling terminal or network flake into the test suite.
- Learnings:
  - For Bubble Tea entry coverage in this repo, stay on the model surface:
    - drive `Update(...)` with `tea.WindowSizeMsg`, `tea.KeyMsg`, and `executeResultMsg`
    - inspect the returned `tea.BatchMsg` to run the command-producing branch without starting a real program
    - keep assertions on transcript, render fallback, and state transitions instead of spinner timing
  - For daemon launchers, a package-local `flag.NewFlagSet` helper with injected `getwd`, handler construction, and serve dispatch is enough to prove root fallback, option normalization, and config wiring without opening a socket.
  - When the wrapper should stay thin, test the seam and leave runtime behavior owned by the shared service or engine layer rather than duplicating integration coverage.
- Scope:
  - Reuse this pattern for future entry-surface regressions in `pkg/cmd`, `cmd/*`, or other thin launchers.
  - Do not use it as a reason to move more runtime logic into the launcher just because the seam is easy to test.

## Promote To

- `docs/.bagakit/memory/howto-learning-20260324-console-simshd-entry-tests.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

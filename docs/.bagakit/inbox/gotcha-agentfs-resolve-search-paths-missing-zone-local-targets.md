---
title: agentfs ResolveSearchPaths missing zone-local targets
kind: gotcha
status: inbox
tags:
  - gotcha
  - agentfs
  - search
  - tests
sources:
  - pkg/adapter/agentfs/filespace_core.go
  - pkg/adapter/agentfs/filespace_core_test.go
created: 2026-03-24T05:02:56Z
---

## Candidate
- Context:
  - While closing `manual-20260324-agentfs-search-path-regressions`, a new direct package regression was added for `ResolveSearchPaths("/knowledge_base/docs/missing.md", false)`.
  - The existing implementation returned the normalized virtual path for a missing file inside an allowed zone instead of returning `No such file or directory`.
- Gotcha:
  - `pkg/adapter/agentfs.(*aiFilesystem).ResolveSearchPaths()` must check host-path existence even after zone resolution succeeds.
  - Allowed-zone membership alone is not enough to treat a target as a valid searchable file.
  - The correct contract matches the other filesystem and mount adapters: missing targets return `No such file or directory`; directories without `-r` return the directory guidance error.
- Why it matters:
  - Returning a missing path as a valid search target leaks a false-positive result into planner-visible search loops.
  - The bug only shows up on missing files inside allowed zones, so zone-boundary tests alone will not catch it.
  - The direct regression belongs in the package-local `agentfs` suite because the behavior is owned by `filespace_core.go`, not by higher integration layers.

## Promote To
- `docs/.bagakit/memory/gotcha-agentfs-resolve-search-paths-missing-zone-local-targets.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

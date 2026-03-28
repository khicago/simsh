---
title: agentfs EditFile single-match contract
kind: gotcha
status: inbox
tags:
  - gotcha
  - agentfs
  - tests
sources:
  - pkg/adapter/agentfs/filespace_core.go
  - pkg/adapter/agentfs/filespace_core_test.go
created: 2026-03-24T04:45:26Z
---

## Candidate
- Context:
  - While adding direct mutator regressions for `manual-20260324-agentfs-mutation-write-limit-regressions`, the first package-local test attempt tried to exercise `EditFile(..., replaceAll=false)` on content with multiple `oldString` matches.
  - `go test ./pkg/adapter/agentfs` failed with `old string appears N times`, which came from the existing production contract rather than a new bug.
- Gotcha:
  - `pkg/adapter/agentfs.(*aiFilesystem).EditFile()` requires exactly one `oldString` match when `replaceAll=false`.
  - If the content contains multiple matches, the method returns `old string appears <n> times` and does not mutate the file.
  - Use a different repeated token when you also need a `replaceAll=true` assertion in the same regression test.
- Why it matters:
  - A test that reuses the same repeated token for both paths will fail for the wrong reason and can look like a production regression even though the API is behaving as designed.
  - Keeping the single-match and replace-all cases separate makes the mutator contract readable and avoids weakening the existing error semantics.

## Promote To
- `docs/.bagakit/memory/gotcha-agentfs-editfile-single-match.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

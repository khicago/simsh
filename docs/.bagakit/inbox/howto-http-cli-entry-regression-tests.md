---
title: Deterministic HTTP and CLI entry-surface regression tests
kind: howto
status: inbox
tags:
  - howto
  - tests
  - httpapi
  - cli
  - long-run
sources:
  - cmd/simsh-cli/main_test.go
  - pkg/service/httpapi/handler_test.go
  - pkg/service/service_test.go
  - pkg/service/httpapi/execute_endpoint.go
  - pkg/service/service.go
created: 2026-03-24T04:15:34Z
---

## Candidate
- Context:
  - The 2026-03-24 long-run coding pass for `manual-20260324-http-cli-entry-regressions` needed deterministic coverage on public entry surfaces without widening the CLI, HTTP, or service abstractions.
- How-to:
  - For `runServe()`, pass `listenAddr="127.0.0.1"` with `rootDir=""` so `http.ListenAndServe` fails immediately with a missing-port error while still printing the banner that reveals cwd root fallback and mount normalization.
  - For `describePathMetaViaLS()`, build a real runtime stack with `EnableTestCorpus=true` and query `/test/core-strict/cases/echo-basic.sh`; this hits a mounted path that `ls -al --fmt json` can describe without unsafe mutation.
  - For `ExecutorService.Execute`, keep the tests on the public wrapper surface: nil-service and nil-engine errors, empty-command fast path, `OpsFactory` error propagation, and one success case using the real builtin registry plus `fs.NewRuntimeOps(...)`.
- Why it worked:
  - Each test stays inside the existing contracts instead of adding test-only seams.
  - The mounted `/test` corpus is stable and already part of the runtime’s supported fixtures, so it is a better fallback target than synthetic mocked metadata.
  - The service wrapper can be validated with the real engine in-process, which closes coverage without touching production code.

## Promote To
- `docs/.bagakit/memory/howto-http-cli-entry-regression-tests.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

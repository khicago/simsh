# BK Execution Handoff

## Run Metadata

- Updated At (UTC): 2026-03-24T11:06:13Z
- Updated By: codex
- Branch: main
- Worktree (optional):

## Current Execution Item

- Execution Item ID: manual-20260324-http-meta-request-edge-hardening
- Source System: manual
- Source Ref: pkg/service/httpapi/execute_endpoint.go
- Title: Close HTTP execute or session metadata edge cases and request parsing gaps
- Status: done

## Completed Scope

- Added HTTP regressions for `/v1/sessions` optional-body parsing: empty body, literal `null`, and invalid JSON.
- Added `/v1/execute` request-validation coverage for invalid JSON, blank commands, and session-bound host-root or profile override rejection.
- Added direct `describePathMeta(...)` assertions for policy-shaped default access, default `dir` or `file` kinds, default capability sets, and read-only write-capability stripping.
- Added explicit-root metadata coverage through both the shell-token extraction helper and an HTTP `include_meta` request that names `/`.
- Fixed the only production defect exposed by the new regressions: `extractAbsPaths()` now preserves explicit `/` tokens instead of dropping them unconditionally.

## Files Changed

- `pkg/service/httpapi/handler_test.go`
  - Added focused regressions for session-body parsing, execute validation branches, metadata defaulting, explicit-root metadata, and helper-level path extraction.
- `pkg/service/httpapi/execute_endpoint.go`
  - Removed the unconditional `/` skip in `extractAbsPaths()` so explicit root paths survive into `meta.paths`.
- `.bagakit/long-run/feature-list.json`
  - Marked `EXEC::manual-20260324-http-meta-request-edge-hardening` as `done` with updated evidence and coverage results.
- `.bagakit/long-run/bk-execution-table.json`
  - Marked `manual-20260324-http-meta-request-edge-hardening` as `done` with concrete evidence from this coding pass.
- `.bagakit/long-run/bk-execution-handoff.md`
  - Replaced the initializer-style plan with this completed coding-pass record.

## Commands Run

```bash
gofmt -w pkg/service/httpapi/handler_test.go pkg/service/httpapi/execute_endpoint.go
go test ./pkg/service/httpapi -run 'TestSessionHandlerOptionalJSONBody|TestExecuteHandlerInvalidJSONBody|TestExecuteHandlerCommandRequired|TestExecuteHandlerIncludeMetaExplicitRootPath|TestDescribePathMetaDefaults|TestExecuteHandlerRejectsSessionOverrides|TestExtractAbsPathsParsesShellStyleTokens'
go test ./pkg/service/httpapi ./pkg/service
go test ./pkg/service/httpapi -cover
```

## Check / Test Outcomes

- `go test ./pkg/service/httpapi -run 'TestSessionHandlerOptionalJSONBody|TestExecuteHandlerInvalidJSONBody|TestExecuteHandlerCommandRequired|TestExecuteHandlerIncludeMetaExplicitRootPath|TestDescribePathMetaDefaults|TestExecuteHandlerRejectsSessionOverrides|TestExtractAbsPathsParsesShellStyleTokens'` exited `0`.
- `go test ./pkg/service/httpapi ./pkg/service` exited `0`.
- `go test ./pkg/service/httpapi -cover` exited `0` and reported `coverage: 79.3% of statements`, up from the `71.1%` initializer baseline.

Acceptance criteria:
- [x] `go test ./pkg/service/httpapi ./pkg/service` passes after the new regressions land.
- [x] `/v1/sessions` accepts empty and literal `null` POST bodies, while invalid JSON still returns `400 invalid json body`.
- [x] `describePathMeta(...)` defaulting branches are directly asserted for access, capabilities, kind, and read-only write-capability stripping.
- [x] An explicit `include_meta` request that names `/` preserves `/` in `meta.paths`, with the production fix kept local to `extractAbsPaths()`.
- [x] `go test ./pkg/service/httpapi -cover` rises above the `71.1%` baseline without regressing the existing success-path and session-lifecycle tests.

## Residual Risks

- `splitShellWords()` is still a lightweight tokenizer, so metadata extraction remains heuristic for more complex shell syntax than the current regression matrix covers.
- `describePathMetaViaLS()` fallback behavior is still only lightly exercised compared with the direct metadata path; this pass intentionally stayed within the planned request-parsing and explicit-root closure.

## Next Run Suggestion

- Start the next single-item pass on `manual-20260324-long-run-temp-space-preflight`.
- Re-run `bash .bagakit/long-run/check_and_resume.sh` first so the orchestrator can resync `next-action.json` and render the next handoff from the updated execution state.

## Next Command

`bash .bagakit/long-run/check_and_resume.sh`

## Response Snapshot

[[BAGAKIT]]
- LongRun: Item=manual-20260324-http-meta-request-edge-hardening; Status=done; Confidence=0.99; Evidence=focused httpapi regressions + `go test ./pkg/service/httpapi ./pkg/service` + `go test ./pkg/service/httpapi -cover` (79.3%); Next=bash .bagakit/long-run/check_and_resume.sh

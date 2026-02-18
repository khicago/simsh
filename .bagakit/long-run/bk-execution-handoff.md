# BK Execution Handoff

## Run Metadata

- Updated At (UTC): 2026-02-18T12:33:30Z
- Updated By: coding
- Branch: feat/f-20260218-path-access-metadata
- Worktree (optional): .worktrees/wt-f-20260218-path-access-metadata

## Current Execution Item

- Execution Item ID: bagakit-ft:f-20260218-path-access-metadata:T-003
- Source System: bagakit-ft
- Source Ref: .bagakit/ft-harness/feats/f-20260218-path-access-metadata/tasks.json
- Feat ID: f-20260218-path-access-metadata
- Task ID: T-003
- Title: /v1/execute include_meta response metadata
- Status: done
- Why This Item Now: With SSOT access/capabilities and `ls -l --fmt json` in place, we can expose opt-in metadata to API callers without duplicating rules.

## Acceptance Criteria

- [x] `/v1/execute` accepts `include_meta=true`.
- [x] Response includes `meta.paths[]` derived from SSOT access/capabilities (via `ls -al --fmt json <path>` introspection).
- [x] `go test ./...` passes in `.worktrees/wt-f-20260218-path-access-metadata`.
- [x] SSOT updated: `tasks.json` includes gate evidence + commit hash; task finished `done`; feat archived.

## Execution Plan

1. Implement include_meta request/response shape in `pkg/service/httpapi/execute_endpoint.go`.
2. Derive `meta.paths[]` by extracting absolute paths from the command line, then running `ls -al --fmt json <path>` inside the same runtime stack.
3. Add an HTTP handler test covering include_meta.
4. Run gofmt + tests, commit, gate, finish, and close feat.

## Files To Touch

- `.worktrees/wt-f-20260218-path-access-metadata/pkg/service/httpapi/execute_endpoint.go`
- `.worktrees/wt-f-20260218-path-access-metadata/pkg/service/httpapi/handler_test.go`
- `.worktrees/wt-f-20260218-path-access-metadata/pkg/builtin/op_listing.go`
- `.bagakit/ft-harness/feats/f-20260218-path-access-metadata/tasks.json`
- `.bagakit/ft-harness/feats/f-20260218-path-access-metadata/state.json`
- `.bagakit/ft-harness/feats/f-20260218-path-access-metadata/summary.md`
- `docs/.bagakit/inbox/decision-f-20260218-path-access-metadata.md`
- `docs/.bagakit/inbox/howto-f-20260218-path-access-metadata-result.md`
- `.bagakit/long-run/bk-execution-handoff.md`

## Commands To Run

```bash
git -C .worktrees/wt-f-20260218-path-access-metadata status -sb
(
  cd .worktrees/wt-f-20260218-path-access-metadata && \
  gofmt -w ./pkg && \
  go test ./...
)
sh .bagakit/long-run/init.sh
```

## Results

- Summary:
  - Added `/v1/execute` `include_meta` opt-in and returned `meta.paths[]`.
  - Implemented meta derivation via runtime `ls -al --fmt json <path>` introspection.
  - Updated ls JSON output to always use absolute `path` (not display path), so `.` resolves to the real directory path.
- Tests:
  - `(cd .worktrees/wt-f-20260218-path-access-metadata && go test ./...)` PASS
- Gate / SSOT:
  - feat archived (`ft_feat_close.sh`)
  - tasks updated with commit hashes:
    - T-001: f81a7f1
    - T-002: 0d88ab9
    - T-003: 48a86f7

## Response Driver Snapshot

```text
[[BAGAKIT]]
- LivingDoc: Followed must-guidebook/must-sop; created/updated architecture doc and regenerated must-sop.
- LongRun: Item=bagakit-ft:f-20260218-path-access-metadata:T-003; Status=done; Evidence=go test ./... + gate logs + tasks.json commit hashes; Next=sh .bagakit/long-run/init.sh
```

## Next Run

- Run: `sh .bagakit/long-run/init.sh`

# Feat Proposal: f-20260224-plan-sync-agentfs-write-limit

## Why
- `docs/first_version_plan.md` 的完成状态与当前实现不一致，容易导致后续排期误判。
- `ExecutionPolicy.MaxWriteBytes` 目前没有联动到 `agentfs` 默认运行时构建路径，导致 write-limited 场景在不同入口行为不一致。

## Goal
- 校准首版计划状态并把ExecutionPolicy.MaxWriteBytes联动到agentfs写入限制，补齐回归验证

## Scope
- In scope:
  - 更新 `docs/first_version_plan.md` 的 Sprint 5 状态描述与后续计划条目。
  - 在 `pkg/fs` 运行时装配层把 `policy.MaxWriteBytes` 传给 `agentfs.Options.WriteLimitedBytes`。
  - 补充/更新测试并跑通 `go test ./...`。
- Out of scope:
  - 新增命令能力、协议字段、外部依赖。
  - 变更 feat/task harness 机制。

## Impact
- Code paths:
  - `docs/first_version_plan.md`
  - `pkg/fs/filesystem_env.go`
  - `pkg/fs/*_test.go`（如需）
- Tests:
  - `go test ./...`
- Rollout notes:
  - 仅影响本地 runtime 装配和文档，不涉及接口破坏。

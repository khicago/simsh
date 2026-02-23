# Feat Proposal: f-20260220-tooling-docs-consistency

## Why
- `man` 文档来源存在双份文件（`pkg/builtin/manuals/*.md` 与 `pkg/builtin/commands/*/manual.md`），已出现漂移（`ls` 手册不一致）。
- `man -v` 会直接暴露 markdown frontmatter，影响可读性和 token 效率。
- README builtin 列表落后于当前实现，且缺少关键诊断工具（`pwd` / `which` / `type` / `rmdir`）。

## Goal
- 按1/2/3顺序完成提交拆分策略、man/manual/README与工具补齐优化，并通过全量校验

## Scope
- In scope:
- 手册单一来源：统一到 `pkg/builtin/commands/*/manual.md`，移除 `pkg/builtin/manuals/*.md`。
- `man` 渐进式输出优化：summary 增加 `Use-When/Avoid-When`；verbose 去除 frontmatter。
- 新增 builtin：`pwd`、`which`、`type`、`rmdir`。
- 新增 builtin：`tree`（支持 `-a` 与 `-L`）。
- 扩展目录删除能力：新增 `Ops.RemoveDir` 并在 localfs/mount router 接入。
- README 与架构文档更新（builtin 列表 + manual SSOT 规则）。
- Out of scope:
- P1/P2 工具批次（`cut/tr/xargs`、`basename/dirname/realpath/stat`）。
- 直接执行任意 bash 脚本（仅讨论可行性与边界，不在本次实现）。
- long-run skill 本身改造。

## Impact
- Code paths:
- `pkg/builtin/*`（命令注册、man 输出、manual embed、新增命令）
- `pkg/contract/integration_contract.go`
- `pkg/engine/{orchestrator.go,virtualfs_bridge.go,engine_test.go}`
- `pkg/adapter/localfs/{adapter.go,adapter_test.go}`
- Tests:
- `go test ./...`
- `staticcheck ./...`
- `feat_task_harness.py validate-harness|diagnose-harness`
- Rollout notes:
- 与现有命令向后兼容；新增命令仅扩展能力，不改变既有命令语义。

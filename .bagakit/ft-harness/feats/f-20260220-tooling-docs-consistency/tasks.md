# Feat Tasks: f-20260220-tooling-docs-consistency

JSON SSOT: `tasks.json`

## Task Checklist
- [x] T-001 Manual/Command 统一优化与能力补齐（manual SSOT + man 渐进式 + pwd/which/type/rmdir + docs/test）

## T-001 Notes
- 完成 `manuals/*.md` 到 `commands/*/manual.md` 单一来源迁移。
- 新增 `tree` / `pwd` / `which` / `type` / `rmdir` 命令与手册。
- 引入 `Ops.RemoveDir`，并在 localfs + mount router 落地。
- 覆盖测试：builtin/engine/localfs + 全量 `go test`、`staticcheck`、harness validate/doctor。

## Status Legend
- todo
- in_progress
- done
- blocked

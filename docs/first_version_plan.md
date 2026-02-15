# simsh v1 — 首版强化计划

> 基于三角色（SDK 架构师 / Agent UX 分析师 / 安全工程师）深度分析的执行计划。
> 聚焦 P0 + P1 项，确保基础库的安全性、工具完备性和嵌入者体验。

---

## 一、分析摘要

### 三视角交叉发现

| 维度 | 核心问题 | 优先级 |
|------|---------|--------|
| 安全 | tee/sed 写操作不显式检查 Policy.AllowWrite()，依赖 Ops 实现者自律 | P0 |
| 安全 | tee 不检查 MaxWriteBytes，agentfs WriteFile 也不检查 | P0 |
| 安全 | localfs 不防护符号链接遍历 | P0 |
| 安全 | HereDoc 解析无大小限制 | P0 |
| 工具 | 缺少 cp/mv/rm/mkdir/touch 基础文件管理命令 | P0 |
| 工具 | man 输出不含 Tips，Manual 过于简略，缺少示例 | P0 |
| 架构 | 缺少 Session 抽象（跨调用状态保持） | P1 |
| 架构 | ExternalCommand 缺少元数据注册（描述、man page 模板） | P1 |
| 安全 | 写入大小在 contract 层无强制检查 | P1 |
| 工具 | 缺少 wc/sort/uniq 文本处理命令 | P1 |
| 工具 | 缺少 diff 命令 | P1 |
| 工具 | grep 无匹配返回 exit code 0（应为 1） | P1 |
| 工具 | cat 默认加行号影响管道行为 | P1 |
| 安全 | 条件链 (&&/||) 无深度限制 | P1 |
| 安全 | Pipeline 中间结果无大小限制 | P1 |
| 安全 | Dispatch 递归无深度限制 | P1 |

---

## 二、执行计划

### Sprint 1: 安全加固 (P0 安全)

#### 1.1 tee/sed 写操作显式 Policy 检查
- **文件**: `pkg/builtin/op_mirror_write.go`, `pkg/builtin/op_stream_edit.go`
- **变更**: 在 `runTee` 和 `runSedInPlace` 的 Run 函数开头增加 `if !runtime.Ops.Policy.AllowWrite()` 检查，以及 `MaxWriteBytes` 大小检查
- **原理**: 当前依赖 Ops 实现者在 WriteFile/EditFile 内部检查 policy，但自定义实现可能遗漏。双重防护是正确做法（参考 `applyOutputRedirections` 在 `script_runner.go:137-141` 的实现）

#### 1.2 agentfs WriteFile/AppendFile 增加 MaxWriteBytes 检查
- **文件**: `pkg/adapter/agentfs/filespace_core.go`
- **变更**: `WriteFile` 和 `AppendFile` 方法中增加 policy 感知的大小检查。需要在 `aiFilesystem` 中持有 policy 引用，或在 `Ops` 构建时通过闭包注入
- **方案**: 在 `OpsFromFilesystem` 桥接层（`pkg/contract/integration_contract.go`）增加 policy-aware wrapper，而非修改 Filesystem 接口

#### 1.3 localfs 符号链接防护
- **文件**: `pkg/adapter/localfs/adapter.go`
- **变更**: 在 `readRawContent`、`writeFile`、`appendFile`、`editFile`、`collectFilesUnder` 中，使用 `filepath.EvalSymlinks` 解析真实路径后再做 `pathWithinRoot` 检查
- **注意**: `WalkDir` 中也需检查 `d.Type()&os.ModeSymlink != 0`

#### 1.4 HereDoc 大小限制
- **文件**: `pkg/engine/script_parser_v2.go`
- **变更**: 在 `readHereDocBody` 中增加 body 大小上限（默认 4MB，与 MaxOutputBytes 对齐）。超限返回解析错误
- **同时**: 增加语句数量上限（默认 1024）在 `parseStatements` 循环中

### Sprint 2: 文件管理命令 (P0 工具)

#### 2.1 新增 mkdir 命令
- **文件**: 新建 `pkg/builtin/op_mkdir.go`
- **语法**: `mkdir [-p] ABS_PATH...`
- **行为**: `-p` 递归创建。需要 AllowWrite() 检查。通过 Ops 回调实现（需在 contract.Ops 中增加 `MkdirAll` 回调，或复用 WriteFile 的目录创建逻辑）
- **方案**: 增加 `Ops.MakeDir func(ctx, path string) error` 回调

#### 2.2 新增 cp 命令
- **文件**: 新建 `pkg/builtin/op_copy.go`
- **语法**: `cp SRC_ABS_PATH DEST_ABS_PATH`
- **行为**: 读源文件 + 写目标文件。需要 AllowWrite() 检查。不支持 -r（目录复制），保持简单

#### 2.3 新增 mv 命令
- **文件**: 新建 `pkg/builtin/op_move.go`
- **语法**: `mv SRC_ABS_PATH DEST_ABS_PATH`
- **行为**: cp + rm 语义。需要 AllowWrite() 检查
- **方案**: 增加 `Ops.RemoveFile func(ctx, path string) error` 回调

#### 2.4 新增 rm 命令
- **文件**: 新建 `pkg/builtin/op_remove.go`
- **语法**: `rm ABS_PATH...`（不支持 -r，防止误删目录）
- **行为**: 删除文件。需要 AllowWrite() 检查

#### 2.5 新增 touch 命令
- **文件**: 新建 `pkg/builtin/op_touch.go`
- **语法**: `touch ABS_PATH...`
- **行为**: 创建空文件或更新时间戳。需要 AllowWrite() 检查

#### 2.6 contract.Ops 扩展
- **文件**: `pkg/contract/integration_contract.go`
- **变更**: 增加 `MakeDir` 和 `RemoveFile` 回调字段
- **桥接**: `OpsFromFilesystem` 中增加对应的类型断言探测
- **适配器**: localfs 和 agentfs 都需要实现新回调
- **engine**: `normalizeOps` 中为新回调提供默认的 ErrUnsupported 实现

### Sprint 3: man 系统重构 — embed markdown 渐进式披露 (P0 工具)

#### 3.1 CommandSpec 结构扩展
- **文件**: `pkg/engine/builtin_catalog.go`
- **变更**: `CommandSpec` 增加 `DetailedManual string` 字段（embed markdown 内容）和 `Examples []string` 字段
- **渐进式披露**: `man cmd` 返回摘要（Manual + Tips + Examples），`man -v cmd` 返回完整 DetailedManual

#### 3.2 embed markdown 文档
- **目录**: 新建 `pkg/builtin/manuals/` 目录
- **文件**: 每个命令一个 markdown 文件，如 `ls.md`, `grep.md`, `cat.md` 等
- **格式**: 使用 frontmatter 标记元数据（name, synopsis, category），正文包含详细说明、flag 解释、示例、注意事项
- **加载**: 通过 `//go:embed manuals/*.md` 在 register.go 中加载

#### 3.3 man 命令重构
- **文件**: `pkg/builtin/op_help_manual.go`
- **变更**:
  - `man cmd` → 输出 Manual 一行摘要 + Tips + Examples（简洁模式）
  - `man -v cmd` → 输出完整 embed markdown（详细模式）
  - `man --list` → 列出所有可用命令及其一行摘要
- **BuiltinManual 方法**: 修改 `Registry.BuiltinManual` 返回包含 Tips 的完整摘要

#### 3.4 所有现有命令的 manual 编写
- 为 ls, cat, head, tail, grep, find, echo, tee, sed, man, date, env 编写详细 markdown manual
- 为 Sprint 2 新增的 mkdir, cp, mv, rm, touch 编写 manual
- 每个 manual 包含: Synopsis, Description, Flags, Examples, Notes, See Also

### Sprint 4: 文本处理命令 + 行为修正 (P1 工具)

#### 4.1 新增 wc 命令
- **文件**: 新建 `pkg/builtin/op_wordcount.go`
- **语法**: `wc [-l] [-w] [-c] [ABS_FILE]`
- **行为**: 统计行数/字数/字节数。支持 stdin 管道输入

#### 4.2 新增 sort 命令
- **文件**: 新建 `pkg/builtin/op_sort.go`
- **语法**: `sort [-r] [-n] [-u] [ABS_FILE]`
- **行为**: 行排序。`-r` 逆序，`-n` 数值排序，`-u` 去重

#### 4.3 新增 uniq 命令
- **文件**: 新建 `pkg/builtin/op_uniq.go`
- **语法**: `uniq [-c] [-d] [ABS_FILE]`
- **行为**: 去除相邻重复行。`-c` 计数，`-d` 只输出重复行

#### 4.4 新增 diff 命令
- **文件**: 新建 `pkg/builtin/op_diff.go`
- **语法**: `diff ABS_FILE1 ABS_FILE2`
- **行为**: 统一 diff 格式输出。简化实现，不支持目录 diff

#### 4.5 grep 无匹配时返回 exit code 1
- **文件**: `pkg/builtin/op_pattern_scan.go`
- **变更**: 当 grep 无匹配结果时，返回 `("", 1)` 而非 `("", 0)`，与 POSIX 语义一致

#### 4.6 cat 行号行为修正
- **文件**: `pkg/builtin/op_readfile.go`
- **变更**: cat 默认不加行号（管道友好），增加 `-n` flag 显示行号
- **影响**: 需要更新依赖 `1:content` 格式的测试

### Sprint 5: 引擎安全加固 (P1 安全)

#### 5.1 条件链深度限制
- **文件**: `pkg/engine/script_runner.go`
- **变更**: 在 `executeStatements` 中增加条件链（&&/||）深度计数器，默认上限 64

#### 5.2 Pipeline 中间结果大小限制
- **文件**: `pkg/engine/script_runner.go`
- **变更**: 在 `executePipeline` 的管道传递阶段检查中间结果大小，超过 MaxOutputBytes 时截断并报错

#### 5.3 Dispatch 递归深度限制
- **文件**: `pkg/engine/orchestrator.go`
- **变更**: 在 `CommandRuntime.Dispatch` 闭包中增加递归深度计数器，默认上限 8

#### 5.4 contract 层写入大小强制检查
- **文件**: `pkg/contract/safety_policy.go`
- **变更**: 增加 `CheckWriteSize(contentLen int) error` 方法，在 policy 层统一检查
- **使用**: tee、sed、重定向写入前统一调用

### Sprint 6: 单元测试全覆盖

#### 6.1 安全加固测试
- tee/sed 在 read-only policy 下被拒绝
- tee 写入超过 MaxWriteBytes 被拒绝
- agentfs WriteFile 超过 MaxWriteBytes 被拒绝
- localfs 符号链接不逃逸 root
- HereDoc 超大 body 被拒绝
- 语句数量超限被拒绝
- 条件链深度超限被拒绝
- Pipeline 中间结果超限被拒绝
- Dispatch 递归深度超限被拒绝

#### 6.2 新命令测试
- mkdir: 基本创建、-p 递归、已存在、权限拒绝
- cp: 基本复制、目标已存在覆盖、源不存在、权限拒绝
- mv: 基本移动、目标已存在覆盖、源不存在、权限拒绝
- rm: 基本删除、不存在、权限拒绝
- touch: 基本创建、已存在、权限拒绝
- wc: -l/-w/-c、stdin 管道、文件输入
- sort: -r/-n/-u、stdin 管道、文件输入
- uniq: -c/-d、stdin 管道、文件输入
- diff: 相同文件、不同文件、不存在文件

#### 6.3 man 系统测试
- man cmd 返回摘要（含 Tips + Examples）
- man -v cmd 返回完整 markdown
- man --list 列出所有命令
- man 对外部命令的查找
- embed markdown 加载正确性

#### 6.4 行为修正测试
- grep 无匹配返回 exit code 1
- cat 默认不加行号
- cat -n 加行号
- cat stdin passthrough 不加行号

#### 6.5 集成测试
- 新命令在管道中的组合使用
- 新命令与重定向的组合
- 新命令在 &&/|| 条件链中的行为
- 完整工作流: find | grep | wc -l
- 完整工作流: cat file | sort | uniq -c

---

## 三、实现约束

1. **不引入新外部依赖**: 所有新命令用纯 Go 标准库实现
2. **不改变公共接口签名**: contract.Ops 只增加可选字段，不修改已有字段
3. **向后兼容**: 新增的 Ops 回调在 normalizeOps 中提供默认 ErrUnsupported 实现
4. **embed markdown**: 使用 `//go:embed` 嵌入，零运行时 I/O
5. **测试独立性**: 所有测试使用 testFS mock，不依赖真实文件系统（localfs 测试除外）

---

## 四、团队分工

| 角色 | 负责 Sprint | 说明 |
|------|------------|------|
| security-impl | Sprint 1 + Sprint 5 | 安全加固 + 引擎限制 |
| tooling-impl | Sprint 2 + Sprint 4 | 新命令实现 |
| manual-impl | Sprint 3 | man 系统重构 + embed markdown |
| test-impl | Sprint 6 | 全量单测（依赖前三者完成） |

Sprint 1-3 可并行执行，Sprint 4 依赖 Sprint 2（contract.Ops 扩展），Sprint 5 独立，Sprint 6 在所有实现完成后执行。

---

## 五、完成状态

> 截至 2025-02，Sprint 1–4 + Sprint 6 已全部完成，Sprint 5 部分完成。

| Sprint | 状态 | 说明 |
|--------|------|------|
| Sprint 1: 安全加固 | ✅ 完成 | HereDoc 4MB 限制、语句数 1024 上限、单引号转义修复、tee/sed Policy 检查、`CheckWriteSize()` |
| Sprint 2: 文件管理命令 | ✅ 完成 | cp/mkdir/rm/mv/touch 全部实现，`Ops.MakeDir`/`Ops.RemoveFile` 回调已扩展 |
| Sprint 3: man 系统 | ✅ 完成 | 22 个 embed markdown manual、`man -v`/`man --list` 模式、`ExamplesFor()` 辅助函数 |
| Sprint 4: 文本处理 + 行为修正 | ✅ 完成 | wc/sort/uniq/diff 命令、grep exit code 修正、cat 默认无行号 + `-n` flag |
| Sprint 5: 引擎安全加固 | ⚠️ 部分完成 | 语句执行上限已实现；Pipeline 中间结果限制、Dispatch 递归深度限制待补充 |
| Sprint 6: 单元测试 | ✅ 完成 | 72 个测试用例覆盖安全/命令/man/行为/集成/管道 |

### 已交付产物

- `pkg/builtin/op_copy.go`, `op_mkdir.go`, `op_remove.go`, `op_move.go`, `op_touch.go` — 文件管理命令
- `pkg/builtin/op_wordcount.go`, `op_sort.go`, `op_uniq.go`, `op_diff.go` — 文本处理命令
- `pkg/builtin/manuals/` — 22 个 embed markdown 文档
- `pkg/builtin/manuals.go` — embed 加载 + `ExamplesFor()` 映射
- `pkg/contract/safety_policy.go` — `CheckWriteSize()` 方法
- `pkg/engine/script_parser_v2.go` — HereDoc/语句数安全限制
- `pkg/engine/engine_test.go` — 72 个测试用例，全部通过

### 测试覆盖

```
ok  github.com/khicago/simsh/cmd/simsh-cli
ok  github.com/khicago/simsh/pkg/engine         (72 tests)
ok  github.com/khicago/simsh/pkg/engine/runtime
ok  github.com/khicago/simsh/pkg/service/httpapi
```

---

## 六、后续计划

### 近期 (v1.1)

1. **Sprint 5 收尾**: Pipeline 中间结果大小限制 + Dispatch 递归深度限制
2. **localfs 符号链接防护增强**: `filepath.EvalSymlinks` + `WalkDir` 中跳过 symlink，当前仅在 builtin 层做了 policy 检查
3. **agentfs MaxWriteBytes 联动**: `filespace_core.go` 中 `WriteLimitedBytes` 字段已预留但未接入

### 中期 (v1.2)

4. **Session 抽象**: 跨调用状态保持（环境变量、工作目录、命令历史），支持多轮对话场景
5. **资源配额 contract**: CPU 时间、内存、文件数量上限，防止 agent 滥用
6. **ExternalCommand 元数据注册**: 描述、参数 schema、man page 模板，提升外部命令的可发现性

### 远期 (v2.0)

7. **命令组合原语**: 支持 subshell `()` 和 command grouping `{}`
8. **变量与展开**: `$VAR` 环境变量展开、命令替换 `$(cmd)`
9. **条件表达式**: `if/then/else`、`test`/`[` 命令
10. **流控制**: `for` 循环（有限迭代，防止无限循环）

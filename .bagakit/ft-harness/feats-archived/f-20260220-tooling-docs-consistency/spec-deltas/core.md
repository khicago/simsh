# Spec Delta: core

## ADDED Requirements

### Requirement: Builtin manual single source of truth
The system SHALL load builtin detailed manuals only from `pkg/builtin/commands/*/manual.md`.

#### Scenario: prevent manual drift
- **WHEN** a command manual is updated
- **THEN** `man -v <cmd>` and registry docs use the same markdown source without dual-file divergence.

### Requirement: man progressive disclosure guardrails
The system SHALL provide concise summary guidance and clean verbose rendering.

#### Scenario: summary mode guidance
- **WHEN** `man <cmd>` is executed
- **THEN** output includes concise `Use-When` and `Avoid-When` guidance for quick decision-making.

#### Scenario: verbose mode readability
- **WHEN** `man -v <cmd>` is executed for markdown manuals with frontmatter
- **THEN** YAML frontmatter is stripped before rendering.

### Requirement: command introspection and directory removal primitives
The runtime SHALL expose command-discovery builtins and a first-class directory-removal callback.

#### Scenario: introspection commands
- **WHEN** `pwd` / `which` / `type` are executed
- **THEN** runtime returns deterministic path/source answers using builtin-first then external lookup order.

#### Scenario: directory tree visualization
- **WHEN** `tree [-a] [-L N] ABS_PATH` is executed
- **THEN** runtime returns an ASCII hierarchy view with optional hidden-file and depth controls.

#### Scenario: directory remove operation
- **WHEN** `rmdir ABS_DIR` is executed
- **THEN** it removes empty directories via `Ops.RemoveDir` and rejects unsupported/mount/unsafe paths consistently.

## MODIFIED Requirements
- `contract.Ops` 增加 `RemoveDir`，并由 `OpsFromFilesystem` 按接口探测接入。
- `engine.normalizeOps` 提供 `RemoveDir` 默认 `ErrUnsupported`。
- mount router 对 `RemoveDir` 施加与 `RemoveFile` 同级的 mount/synthetic-prefix 拒绝规则。

## REMOVED Requirements
- duplicated manual source under `pkg/builtin/manuals/*.md`.

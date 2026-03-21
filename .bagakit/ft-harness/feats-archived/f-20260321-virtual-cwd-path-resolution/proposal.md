# Feat Proposal: f-20260321-virtual-cwd-path-resolution

## Why
- Absolute-path-only behavior is safe but not natural enough for agent work, especially when tools, skills, and docs commonly refer to neighboring resources through relative paths.
- The kernel still lacks an explicit model for session-local `cwd`, relative path resolution, and how those semantics should interact with mounts and capability restrictions.

## Goal
- Define and implement an explicit session-local virtual cwd and unified path resolution model that supports relative paths without weakening mount and capability semantics.

## Scope
- In scope:
- session-local virtual `cwd`
- unified path resolution to normalized virtual absolute paths
- explicit handling of relative-path flows for file-oriented builtins
- preserving mount and synthetic path capability limits after resolution
- Out of scope:
- host-shell `cwd` inheritance semantics
- document-relative or skill-relative reference semantics that belong to higher-level loaders
- broad trace redesign except where needed to record resolved execution context

## Impact
- Code paths:
- `pkg/contract`
- `pkg/engine/runtime`
- file-oriented builtins such as `pwd`, `ls`, `cat`, `cp`, `mv`, `rm`, `touch`, and related helpers
- Tests:
- session-level `cwd` tests
- relative-path resolution tests
- mount/synthetic capability enforcement through relative-path access
- Rollout notes:
- this feat should start only after boundary enforcement is hardened enough to trust the resolved absolute path as the true capability subject

# Feat Proposal: f-20260217-vfs-mount-ancestors

## Why
- Problem/opportunity this feat addresses.

## Goal
- 允许显式访问 mount 的父目录（如 /sys），并对 mount/synthetic 路径统一施加不可变写保护（rm/mv/cp/mkdir/重定向等），避免 mv 半写入。

## Scope
- In scope:
- Out of scope:

## Impact
- Code paths:
- Tests:
- Rollout notes:

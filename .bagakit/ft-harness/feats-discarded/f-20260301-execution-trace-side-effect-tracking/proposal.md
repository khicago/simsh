# Feat Proposal: f-20260301-execution-trace-side-effect-tracking

## Why
- Agents need path-level side effects, deny reasons, and timing/resource summaries for planning, but current audit hooks only emit coarse command start/end/error events.
- Without a first-class trace model, adapters would either scrape logs or invent incompatible side-effect trackers.

## Goal
- Track path-level side effects, denials, timing, and resource summaries for builtins and external commands as ExecutionTrace output.

## Scope
- In scope:
- trace collection for builtin file operations and redirections
- external command trace categories at the same abstraction level as builtins
- duration/resource summary fields and denied-path reporting
- Out of scope:
- session lifecycle primitives
- adapter-side memory observation rules
- reference workload implementation

## Impact
- Code paths:
- `pkg/engine`
- `pkg/contract`
- `pkg/builtin`
- `pkg/service/httpapi`
- Tests:
- side-effect trace coverage for read/write/edit/remove flows
- deny-path and output-limit trace tests
- external command trace parity tests
- Rollout notes:
- trace fields must stay directly consumable by higher-level planners instead of degenerating into audit-log copies

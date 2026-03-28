# simsh-native Reference Validation

This benchmark pack is the first-pass P4 reference-validation harness for
`simsh`.

It exists to answer one question:

- does the current kernel measurably support realistic agent file workflows
  well enough to justify the abstraction, not only the unit tests?

Current scenario classes:
- relative path navigation
- inspect/edit/write file loops
- mount and synthetic capability boundaries
- command namespace consistency
- trace-consumable planning
- adapter-backed projection and managed-memory lifecycle, including projection invalidation, refresh, workflow-transition provenance, and structural per-item projection materialization/failure-state assertions for docs/resources/skills (not message-only checks)
  plus explicit `/skills` selection-provenance assertions (scope, mode, and loser/ineligible reason) and deterministic precedence outcomes
  plus structured control-plane audit coverage via `/memory/skills_audit.json` and `/memory/skills_audit.md` so every add/update/remove event, visibility timing, and `/memory/summary.md` alignment is machine-checked
  plus a generic curated-memory entry assertion with explicit source-path references
  plus structured projection metrics (`/memory/projection_metrics.json`/`.md`) and denial surfaces (`/memory/denials.json`/`.md`) that prove projection generation, freshness/materialization counts, and denied-path metadata stay aligned with the control-plane audit
- cancel and timeout interruption

Current metric gates:
- trace completeness `>= 0.90`
- session success `>= 0.80`
- reviewable patch latency median `<= 15m`
- async completion success `>= 0.60`

Metric note:
- `trace_completeness` only counts trace assertions.
- Scenario/business assertions are tracked separately inside each scenario report and do not dilute the trace gate.

Run it with:

```bash
go run ./benchmarks/simsh_native_reference
```

Write a report to disk with:

```bash
go run ./benchmarks/simsh_native_reference -out benchmarks/simsh_native_reference/reports/baseline-latest.json
```

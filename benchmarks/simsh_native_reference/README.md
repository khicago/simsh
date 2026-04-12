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
- adapter composition/evolution stress: a harder multi-step workload that proves those same machine-readable truth surfaces remain aligned together across control-plane mutation, invalidation/refresh, denial accumulation, and checkpoint/close/resume, without introducing new adapter nouns
- resource-set adapter seam: exercises the smaller `resourceset` adapter by reading a resource, forcing a denial, and decoding its minimal `/memory` views before and after the checkpoint/close/resume cycle so this simpler shape proves the same seam without inheriting the full reference model.
- cancel and timeout interruption

Current metric gates:
- trace completeness `>= 0.90`
- session success `>= 0.80`
- reviewable patch latency median `<= 15m`
- async completion success `>= 0.60`

Metric note:
- `trace_completeness` only counts trace assertions.
- Scenario/business assertions are tracked separately inside each scenario report and do not dilute the trace gate.

External benchmark mapping:
- `benchmarks/external_mapping/` is the downstream evaluation-feasibility layer for this native suite.
- Stable native scenario ids and categories remain the primary SSOT.
- External mapping artifacts classify each native scenario as `as_is`, `translated`, or `excluded` for families such as Terminal-Bench and SWE-bench-Live without mutating the native suite to look more benchmark-compatible.
- `benchmarks/terminal_bench_compare/` is the next downstream layer: a lightweight comparison/export prototype that consumes the native report plus Terminal-Bench mapping artifacts without becoming a second benchmark suite.
- `benchmarks/paired_uplift/` is the next proof layer after that: it holds the paired task manifest, budgets, and deterministic probe agent fixed while comparing full `simsh` against one repo-controlled thin baseline substrate.
- `benchmarks/evidence_manifest.json` is the checked-in proof-surface SSOT for the current benchmark evidence set, canonical refresh commands, and artifact-path breadcrumbs across these layers.

Run it with:

```bash
go run ./benchmarks/simsh_native_reference
```

Write a report to disk with:

```bash
go run ./benchmarks/simsh_native_reference -out benchmarks/simsh_native_reference/reports/baseline-latest.json
```

Latest checked-in full baseline used by the Terminal-Bench comparison prototype:

```text
benchmarks/simsh_native_reference/reports/baseline-20260404.json
```

Refresh the native baseline and downstream Terminal-Bench comparison artifacts with:

```bash
make benchmark-refresh
```

Run the paired uplift proof harness with:

```bash
make benchmark-uplift
```

This checked-in baseline is a freshness snapshot, not a byte-stable golden file; `generated_at` and duration-derived fields are expected to change when refreshed.

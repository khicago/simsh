# Paired A/B Uplift Proof Harness

This directory is the K-030 proof layer for `simsh`.

It answers a narrower question than the native benchmark suite or the
Terminal-Bench comparison prototype:

- with the agent, paired tasks, and budgets held fixed, does `simsh` improve
  outcomes relative to one thinner runtime substrate?

## Design Contract

This harness stays downstream from:

- `benchmarks/internal/scenarios/`
- `benchmarks/simsh_native_reference/`
- `benchmarks/external_mapping/`
- `docs/architecture-paired-ab-uplift-proof-harness.md`

It does **not**:

- create a second native scenario catalog
- compare against an ambient host shell
- adopt Terminal-Bench or SWE-bench wholesale
- mutate native benchmark semantics to make `simsh` look better

## Scope SSOT

`task_set.json` is the checked-in scope SSOT for the first cut.

Current design rules:

- one deterministic probe agent
- one repo-controlled thin baseline substrate
- native scenario ids stay canonical
- pair seed, run order, and budgets are explicit per task

## Output Split

This harness keeps the same “freshness snapshot vs downstream artifact” split as
the existing comparison layers.

Checked-in outputs:

- `reports/raw-baseline-20260406.json`
  - freshness snapshot of paired runs
- `reports/paired-baseline-20260406.json`
  - machine-readable aggregate artifact derived from the snapshot
- `reports/paired-baseline-20260406.md`
  - human-readable summary derived from the same snapshot
- `reports/paired-baseline-20260406.failures.json`
  - machine-readable failure taxonomy derived from the same snapshot

## Run It

Generate a fresh paired snapshot plus downstream reports with:

```bash
go run ./benchmarks/paired_uplift
```

Or via the repository target:

```bash
make benchmark-uplift
```

The raw snapshot is a freshness artifact, so `generated_at` and duration fields
are expected to move.
The aggregate JSON/Markdown pair and failure taxonomy are deterministic
downstream renderings of that snapshot.

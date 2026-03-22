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
- trace-consumable planning
- cancel and timeout interruption

Current metric gates:
- trace completeness `>= 0.90`
- session success `>= 0.80`
- reviewable patch latency median `<= 15m`
- async completion success `>= 0.60`

Run it with:

```bash
go run ./benchmarks/simsh_native_reference
```

Write a report to disk with:

```bash
go run ./benchmarks/simsh_native_reference -out benchmarks/simsh_native_reference/reports/baseline-latest.json
```

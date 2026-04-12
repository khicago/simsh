---
name: mounts
synopsis: "mounts [--fmt text|json]\nmounts refresh [--require-narrow] MOUNT_POINT..."
category: introspection
---

# mounts -- show active mount contracts and run explicit scoped refresh

## SYNOPSIS

    mounts [--fmt text|json]
    mounts refresh [--require-narrow] MOUNT_POINT...

## DESCRIPTION

Display the active virtual mounts currently visible to the runtime, including
their normalized mount profile, declared SLO fields, and whether refresh/stats
capabilities exist.

`mounts refresh` is the explicit refresh control-plane entry for mounts that
implement the optional refresh capability. It remains target-scoped and
budgeted; the command does not make ordinary reads trigger hidden refresh
behavior.

## FLAGS

- `--fmt text|json` -- Output format. `json` emits machine-readable mount
  contract records.
- `--require-narrow` -- Require refresh requests to name strict descendant
  targets below the mount root instead of allowing a mount-root or
  adapter-defined broader refresh.

## EXAMPLES

Show a concise text summary:

    mounts

Show structured mount metadata:

    mounts --fmt json

Refresh one explicit mount target:

    mounts refresh <mount-target>

Require an explicit narrow refresh target below the mount root:

    mounts refresh --require-narrow <mount-target>/<child>

## NOTES

- `mounts` status output is read-only.
- `mounts refresh` is explicit control-plane work and stays subject to refresh
  scope/budget/refusal rules; it is not a hidden side effect of status reads.
- `requested_targets` records what the caller asked for. `effective_targets`
  records the actual refresh scope after contract checks. Narrow refresh must
  keep that effective scope narrow instead of silently broadening it.
- The output is intended to make mount point, latency class, consistency, and
  declared SLO budgets visible before running high-fanout workloads.
- `has_refresher`, `has_status`, and `has_stats` only indicate capability
  presence.
- `runtime_status` is optional runtime truth when a mount exposes it. The
  command does not infer stale/materialization state by scanning the mounted
  tree, and transport/status lookup failures remain separate from mount
  freshness/materialization truth.

## SEE ALSO

man, ls, tree

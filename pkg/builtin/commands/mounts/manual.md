---
name: mounts
synopsis: "mounts [--fmt text|json]"
category: introspection
---

# mounts -- show active mount contracts and optional runtime status

## SYNOPSIS

    mounts [--fmt text|json]

## DESCRIPTION

Display the active virtual mounts currently visible to the runtime, including
their normalized mount profile, declared SLO fields, and whether refresh/stats
capabilities exist.

## FLAGS

- `--fmt text|json` -- Output format. `json` emits machine-readable mount
  contract records.

## EXAMPLES

Show a concise text summary:

    mounts

Show structured mount metadata:

    mounts --fmt json

## NOTES

- This command is read-only. It does not trigger refresh or mutate mount state.
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

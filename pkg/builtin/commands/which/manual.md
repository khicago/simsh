---
name: which
synopsis: "which [--fmt json] COMMAND..."
category: navigation
---

# which -- resolve command paths

## SYNOPSIS

    which [--fmt json] COMMAND...

## DESCRIPTION

Resolve command names to executable paths inside the virtual runtime.
Lookup order is aliases first, then system builtins under `/sys/bin`, then
custom external commands under `/bin`.

Default output stays one resolved path per line. Use `--fmt json` when you
want a machine-readable lookup summary.

## FLAGS

- `--fmt json` -- Emit lookup results as JSON instead of path-per-line text.

## EXAMPLES

Resolve a builtin:

    which ls

Resolve an external command:

    which report_tool

Resolve multiple commands:

    which ls report_tool

Machine-readable lookup:

    which --fmt json ls report_tool

## NOTES

- `--fmt json` is the only supported flag.
- Accepts command names, absolute command paths, or relative command paths that resolve under `/sys/bin` or `/bin`.
- Alias results are rendered as `alias name='expanded command ...'`.
- Returns non-zero when any requested command is not found.

## SEE ALSO

type, env, man

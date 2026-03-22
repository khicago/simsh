---
name: env
synopsis: "env [--json] [--split] [KEY]"
category: system
---

# env -- display environment variables

## SYNOPSIS

    env [--json] [--split] [KEY]

## DESCRIPTION

Display environment variables. With no arguments, lists all variables in
`KEY=VALUE` format sorted alphabetically. With a KEY argument, displays
only that variable.

The default text output stays shell-friendly. Use `--json` for machine
consumption or `--split` when a list-like variable such as `PATH` is easier
to read one element per line.

## FLAGS

- `--json` -- Emit a machine-readable environment object.
- `--split` -- Split one list-like variable into one item per line. Requires exactly one KEY.

## EXAMPLES

List all environment variables:

    env

Show the PATH variable:

    env PATH

Show PATH as one entry per line:

    env --split PATH

Show PATH as JSON:

    env --json PATH

## NOTES

- Exposes PATH by default, plus exported vars loaded from configured rc files.
- PATH includes `/sys/bin` (builtins) and `/bin` (external commands).
- `export PATH=...` in rc overrides default PATH rendering.
- Returns empty output if the requested key does not exist.
- The first list-like variable with special handling is `PATH`.
- `--json` keeps the same underlying values as the default text mode.

## SEE ALSO

man, date

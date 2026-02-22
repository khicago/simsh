---
name: env
synopsis: "env [KEY]"
category: system
---

# env -- display environment variables

## SYNOPSIS

    env [KEY]

## DESCRIPTION

Display environment variables. With no arguments, lists all variables in
`KEY=VALUE` format sorted alphabetically. With a KEY argument, displays
only that variable.

## EXAMPLES

List all environment variables:

    env

Show the PATH variable:

    env PATH

## NOTES

- Exposes PATH by default, plus exported vars loaded from configured rc files.
- PATH includes `/sys/bin` (builtins) and `/bin` (external commands).
- `export PATH=...` in rc overrides default PATH rendering.
- Returns empty output if the requested key does not exist.

## SEE ALSO

man, date

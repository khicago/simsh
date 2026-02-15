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

- Currently exposes the PATH variable showing the command search order.
- PATH includes `/sys/bin` (builtins) and `/bin` (external commands).
- Returns empty output if the requested key does not exist.

## SEE ALSO

man, date

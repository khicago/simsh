---
name: which
synopsis: "which COMMAND..."
category: navigation
---

# which -- resolve command paths

## SYNOPSIS

    which COMMAND...

## DESCRIPTION

Resolve command names to executable paths inside the virtual runtime.
Lookup order is aliases first, then system builtins under `/sys/bin`, then
custom external commands under `/bin`.

## EXAMPLES

Resolve a builtin:

    which ls

Resolve an external command:

    which report_tool

Resolve multiple commands:

    which ls report_tool

## NOTES

- Does not support flags.
- Accepts command names, absolute command paths, or relative command paths that resolve under `/sys/bin` or `/bin`.
- Alias results are rendered as `alias name='expanded command ...'`.
- Returns non-zero when any requested command is not found.

## SEE ALSO

type, env, man

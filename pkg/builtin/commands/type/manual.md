---
name: type
synopsis: "type COMMAND..."
category: navigation
---

# type -- describe command resolution

## SYNOPSIS

    type COMMAND...

## DESCRIPTION

Describe how each command resolves in the runtime, including its resolved path
and whether it is an alias, builtin, or external command.

## EXAMPLES

Inspect a builtin:

    type ls

Inspect an external command:

    type report_tool

Inspect by absolute command path:

    type /sys/bin/ls

## NOTES

- Lookup order matches execution order: aliases first, then system builtins (`/sys/bin`), then custom externals (`/bin`).
- Does not support flags.
- Returns non-zero when any requested command is not found.

## SEE ALSO

which, env, man

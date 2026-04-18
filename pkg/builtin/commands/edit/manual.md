---
name: edit
synopsis: "edit [--json] [--confirm] [--all] [--count] --old OLD [--new NEW] [--] PATH"
category: mutation
---

# edit -- unique snippet replace

## SYNOPSIS

    edit --old OLD --new NEW PATH
    edit --json --old OLD --new NEW PATH
    edit --all --old OLD --new NEW PATH
    edit --count --old OLD PATH

## DESCRIPTION

Replace a literal snippet in one file. This is the agent-facing edit primitive:
the default is a unique match, so an ambiguous snippet fails instead of
silently changing the wrong copy.

`--count` reports how many times `OLD` occurs, with line numbers, and does not write.
If a snippet matches more than once, `edit` fails and lists the matching lines.

## FLAGS

- `--old OLD` -- Literal snippet to find. Required and must not be empty.
- `--new NEW` -- Replacement text. Required unless `--count` is set.
- `--all` -- Replace every match. Without this flag, more than one match is an error.
- `--count` -- Print the match count and leave the file unchanged.
- `--json` -- Emit a machine-readable summary.
- `--confirm` -- Print a one-line success summary. Default mutation success is silent.
- `--` -- Stop option parsing so a relative path may begin with `-`.

Use `--old=VALUE` or `--new=VALUE` when a snippet begins with `--` and would
otherwise be interpreted as an option.

## EXAMPLES

Replace a unique snippet:

    edit --old TODO --new DONE /task_outputs/notes.md

Inspect matches without writing:

    edit --count --old foo /task_outputs/dup.txt

Replace flag-like text:

    edit --old=--json --new=--output /task_outputs/args.txt

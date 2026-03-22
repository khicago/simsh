---
name: touch
synopsis: "touch [--json] PATH..."
category: file-management
---

# touch -- create empty files

## SYNOPSIS

    touch [--json] PATH...

## DESCRIPTION

Create one or more empty files. If the file already exists, it is not
modified.

Default success output stays silent. Use `--json` when you want explicit
created vs already-existing results.

## FLAGS

- `--json` -- Emit machine-readable status entries for each requested path.

## EXAMPLES

Create a single file:

    touch /task_outputs/notes.md

Create multiple files:

    touch /task_outputs/a.txt /task_outputs/b.txt

Emit JSON status:

    touch --json /task_outputs/notes.md

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Does not modify existing files.
- Status values in `--json` output are `created` and `already_exists`.

## SEE ALSO

mkdir, tee, rm

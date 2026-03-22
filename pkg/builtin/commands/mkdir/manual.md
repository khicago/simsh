---
name: mkdir
synopsis: "mkdir [--confirm] [--json] [-p] PATH..."
category: file-management
---

# mkdir -- create directories

## SYNOPSIS

    mkdir [--confirm] [--json] [-p] PATH...

## DESCRIPTION

Create one or more directories. By default, parent directories must already
exist. Use `-p` to create intermediate directories as needed.

## FLAGS

- `--confirm` -- Emit one low-noise confirmation line per requested path.
- `--json` -- Emit machine-readable path status entries.
- `-p` -- Create parent directories as needed. No error if the directory already exists.

## EXAMPLES

Create a single directory:

    mkdir /task_outputs/reports

Create nested directories:

    mkdir -p /task_outputs/2026/01/data

Create multiple directories:

    mkdir /task_outputs/logs /task_outputs/cache

Emit confirmation lines:

    mkdir --confirm /task_outputs/reports

Emit JSON status:

    mkdir --json -p /task_outputs/cache

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Without `-p`, fails if the parent directory does not exist.
- Default success output stays silent.
- `--confirm` and `--json` report status for the requested target paths only.
- Status values are `created` and `exists`.

## SEE ALSO

ls, cp, rm

---
name: rmdir
synopsis: "rmdir [--confirm] [--json] PATH..."
category: file-management
---

# rmdir -- remove empty directories

## SYNOPSIS

    rmdir [--confirm] [--json] PATH...

## DESCRIPTION

Remove one or more empty directories.

Default success output stays silent. Use `--confirm` for a low-noise text
acknowledgement or `--json` for a machine-readable removal summary.

## FLAGS

- `--confirm` -- Emit one confirmation line per removed directory.
- `--json` -- Emit machine-readable removal status entries.

## EXAMPLES

Remove one empty directory:

    rmdir /task_outputs/cache

Remove multiple empty directories:

    rmdir /task_outputs/a /task_outputs/b

Emit confirmation:

    rmdir --confirm /task_outputs/cache

Emit JSON status:

    rmdir --json /task_outputs/cache

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Only empty directories can be removed.
- Write operations are still restricted by policy and mount immutability rules.
- `--confirm` and `--json` only report successful removals; failures still follow the normal non-zero exit code path.

## SEE ALSO

mkdir, rm

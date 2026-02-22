---
name: rmdir
synopsis: "rmdir ABS_DIR..."
category: file-management
---

# rmdir -- remove empty directories

## SYNOPSIS

    rmdir ABS_DIR...

## DESCRIPTION

Remove one or more empty directories.

## EXAMPLES

Remove one empty directory:

    rmdir /task_outputs/cache

Remove multiple empty directories:

    rmdir /task_outputs/a /task_outputs/b

## NOTES

- All paths must be absolute.
- Only empty directories can be removed.
- Write operations are still restricted by policy and mount immutability rules.

## SEE ALSO

mkdir, rm

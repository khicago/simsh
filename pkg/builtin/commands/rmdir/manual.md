---
name: rmdir
synopsis: "rmdir PATH..."
category: file-management
---

# rmdir -- remove empty directories

## SYNOPSIS

    rmdir PATH...

## DESCRIPTION

Remove one or more empty directories.

## EXAMPLES

Remove one empty directory:

    rmdir /task_outputs/cache

Remove multiple empty directories:

    rmdir /task_outputs/a /task_outputs/b

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Only empty directories can be removed.
- Write operations are still restricted by policy and mount immutability rules.

## SEE ALSO

mkdir, rm

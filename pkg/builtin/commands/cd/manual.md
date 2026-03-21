---
name: cd
synopsis: "cd [PATH]"
category: navigation
---

# cd -- change virtual working directory

## SYNOPSIS

    cd [PATH]

## DESCRIPTION

Change the session-local virtual working directory used to resolve relative
paths. If no path is provided, `cd` returns to the virtual root.

## EXAMPLES

Change into task outputs:

    cd /task_outputs

Move relative to the current working directory:

    cd ../knowledge_base

Return to the virtual root:

    cd

## NOTES

- Relative paths are resolved against the current virtual working directory.
- The target must resolve to a directory.
- Mount-backed and synthetic paths keep their existing capability limits after `cd`.

## SEE ALSO

pwd, ls, find

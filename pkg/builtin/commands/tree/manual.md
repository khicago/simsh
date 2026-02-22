---
name: tree
synopsis: "tree [-a] [-L N] [ABS_PATH...]"
category: navigation
---

# tree -- display directory structure

## SYNOPSIS

    tree [-a] [-L N] [ABS_PATH...]

## DESCRIPTION

Render directory contents as an ASCII tree. If no path is provided, it starts
from the virtual runtime root.

## FLAGS

- `-a` -- Include hidden entries (names starting with `.`).
- `-L N` -- Limit recursion depth to a non-negative integer.

## EXAMPLES

Show root tree:

    tree /

Limit depth to 2:

    tree -L 2 /task_outputs

Include hidden files:

    tree -a /knowledge_base

## NOTES

- All paths must be absolute.
- File targets are printed as a single line path.
- `-L 0` prints only the root line for directory targets.

## SEE ALSO

ls, find

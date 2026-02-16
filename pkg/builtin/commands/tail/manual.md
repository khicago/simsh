---
name: tail
synopsis: "tail [-n N|-N] [ABS_FILE]"
category: navigation
---

# tail -- display last N lines

## SYNOPSIS

    tail [-n N|-N] [ABS_FILE]
    COMMAND | tail [-n N|-N]

## DESCRIPTION

Output the last N lines of a file or stdin. Default is 10 lines.

## FLAGS

- `-n N` -- Number of lines to display. Accepts `-n N`, `-nN`, `-n=N`, or `-N` shorthand.

## EXAMPLES

Show last 10 lines (default):

    tail /task_outputs/report.md

Show last 5 lines:

    tail -n 5 /task_outputs/report.md

Shorthand form:

    tail -20 /knowledge_base/data.txt

From a pipeline:

    cat /task_outputs/log.txt | tail -n 3

## NOTES

- N must be a non-negative integer.
- All paths must be absolute.
- Reads from stdin when no file path is provided.

## SEE ALSO

head, cat

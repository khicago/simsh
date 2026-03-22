---
name: head
synopsis: "head [-n N|-N] [PATH]"
category: navigation
---

# head -- display first N lines

## SYNOPSIS

    head [-n N|-N] [PATH]
    COMMAND | head [-n N|-N]

## DESCRIPTION

Output the first N lines of a file or stdin. Default is 10 lines.

## FLAGS

- `-n N` -- Number of lines to display. Accepts `-n N`, `-nN`, `-n=N`, or `-N` shorthand.

## EXAMPLES

Show first 10 lines (default):

    head /task_outputs/report.md

Show first 5 lines:

    head -n 5 /task_outputs/report.md

Shorthand form:

    head -20 /knowledge_base/data.txt

From a pipeline:

    cat /task_outputs/log.txt | head -n 3

## NOTES

- N must be a non-negative integer.
- Paths may be absolute or relative to the current virtual working directory.
- Reads from stdin when no file path is provided.

## SEE ALSO

tail, cat

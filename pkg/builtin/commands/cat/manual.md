---
name: cat
synopsis: "cat [-n] PATH"
category: navigation
---

# cat -- display file contents

## SYNOPSIS

    cat [-n] PATH
    COMMAND | cat

## DESCRIPTION

Display the contents of a file or pass stdin through. When given a file path,
reads and outputs its full content. When used with no arguments in a pipeline,
passes stdin through unchanged.

By default, output has no line numbers. Use `-n` to prepend line numbers.

## FLAGS

- `-n` -- Number output lines. Each line is prefixed with its 1-based line number followed by a colon.

## EXAMPLES

Display a file:

    cat /knowledge_base/readme.md

Display a file with line numbers:

    cat -n /knowledge_base/readme.md

Pass stdin through a pipeline:

    echo "hello" | cat

Use with grep to show context:

    cat /task_outputs/log.txt | grep ERROR

## NOTES

- Expects exactly one file path, or stdin with no arguments.
- The `-n` flag produces format: `LINENO:content`
- Paths may be absolute or relative to the current virtual working directory.

## SEE ALSO

head, tail, grep

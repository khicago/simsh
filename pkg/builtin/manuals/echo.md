---
name: echo
synopsis: "echo [ARGS...]"
category: text-processing
---

# echo -- output text

## SYNOPSIS

    echo [ARGS...]

## DESCRIPTION

Write arguments to standard output, separated by spaces. Commonly used to
produce text for pipelines or to write content via `tee`.

## EXAMPLES

Simple output:

    echo hello world

Write content to a file via tee:

    echo "report data" | tee /task_outputs/report.txt

Use in a pipeline:

    echo "line1\nline2" | grep "line1"

Empty output:

    echo

## NOTES

- Arguments are joined with a single space.
- No trailing newline is appended beyond the joined text.
- Does not interpret escape sequences.

## SEE ALSO

tee, cat

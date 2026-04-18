---
name: basename
synopsis: "basename [--] PATH"
category: navigation
---

# basename -- final path component

## SYNOPSIS

    basename PATH

## DESCRIPTION

Print the final component of a resolved virtual path. Relative operands are
resolved against the session working directory first.

Use `--` before a relative path that begins with `-`.

## EXAMPLES

    basename /task_outputs/report.md

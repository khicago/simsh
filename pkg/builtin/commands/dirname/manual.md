---
name: dirname
synopsis: "dirname [--] PATH"
category: navigation
---

# dirname -- parent path

## SYNOPSIS

    dirname PATH

## DESCRIPTION

Print the parent of a resolved virtual path. Relative operands are resolved
against the session working directory first.

Use `--` before a relative path that begins with `-`.

## EXAMPLES

    dirname /task_outputs/report.md

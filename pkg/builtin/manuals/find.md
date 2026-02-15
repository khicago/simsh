---
name: find
synopsis: "find [ABS_DIR] -name PATTERN [-o -name PATTERN ...] [-exec CMD {} ';'|+]"
category: search
---

# find -- find files by name pattern

## SYNOPSIS

    find [ABS_DIR] -name PATTERN [-o -name PATTERN ...] [-exec CMD {} ';'|+]

## DESCRIPTION

Recursively search for files matching one or more name patterns under the given
directory. If no directory is specified, searches from the virtual root.

Patterns support shell glob characters: `*`, `?`, `[...]`.

The optional `-exec` clause runs a command for each matched file (with `;`)
or batches all matches into one invocation (with `+`).

## FLAGS

- `-name PATTERN` -- Match files whose basename matches the glob pattern.
- `-o` -- OR operator to combine multiple `-name` patterns.
- `-exec CMD {} ';'` -- Run CMD once per matched file, replacing `{}` with the file path.
- `-exec CMD {} +` -- Run CMD once with all matched files appended.

## EXAMPLES

Find all markdown files:

    find / -name "*.md"

Find in a specific directory:

    find /task_outputs -name "*.json"

Multiple patterns with OR:

    find /knowledge_base -name "*.md" -o -name "*.txt"

Execute a command on each match:

    find /task_outputs -name "*.log" -exec cat {} ;

Batch execution:

    find /knowledge_base -name "*.md" -exec grep "TODO" {} +

## NOTES

- All paths must be absolute.
- Without `-name`, matches all files (`*`).
- Only one `-exec` clause is supported per invocation.

## SEE ALSO

ls, grep

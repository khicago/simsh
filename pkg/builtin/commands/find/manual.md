---
name: find
synopsis: "find [DIR] -name PATTERN [-o -name PATTERN ...] [--fmt jsonl] [-exec CMD {} ';'|+]"
category: search
---

# find -- find files by name pattern

## SYNOPSIS

    find [DIR] -name PATTERN [-o -name PATTERN ...] [--fmt jsonl] [-exec CMD {} ';'|+]

## DESCRIPTION

Recursively search for files matching one or more name patterns under the given
directory. The directory may be absolute or relative to the current virtual
working directory. If no directory is specified, `find` searches from the
current virtual working directory.

Patterns support shell glob characters: `*`, `?`, `[...]`.

The optional `-exec` clause runs a command for each matched file (with `;`)
or batches all matches into one invocation (with `+`).

## FLAGS

- `-name PATTERN` -- Match files whose basename matches the glob pattern.
- `-o` -- OR operator to combine multiple `-name` patterns.
- `--fmt jsonl` -- Emit flat JSONL path records instead of the default text stream.
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

Machine-readable JSONL output:

    find /task_outputs -name "*.json" --fmt jsonl

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Without `-name`, matches all files (`*`).
- Only one `-exec` clause is supported per invocation.
- Default output remains one path per line for pipeline composability.
- `--fmt jsonl` is the explicit structured mode for agent parsing and downstream tooling.
- `--fmt jsonl` is not supported together with `-exec` in the current implementation.

## SEE ALSO

ls, grep

---
name: grep
synopsis: "grep [-E|-F] [-r] [-l] [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH]"
category: search
---

# grep -- search file contents for patterns

## SYNOPSIS

    grep [-E|-F] [-r] [-l] [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH]
    COMMAND | grep [-E|-F] [-l] [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN

## DESCRIPTION

Search for lines matching PATTERN in a file, directory, or stdin. By default
uses fixed-string matching. Use `-E` for regex matching.

Output format includes file path and line number when searching files:
`filepath:lineno:line` for matches, `filepath-lineno:line` for context lines.

When searching stdin, output is: `lineno:line` for matches.

## FLAGS

- `-E` -- Use extended regular expression matching.
- `-F` -- Use fixed-string matching (default).
- `-r` -- Recursive search through directory contents.
- `-l` -- List only file paths that contain a match (no line content).
- `-A N` -- Print N lines after each match.
- `-B N` -- Print N lines before each match.
- `-C N` -- Print N lines before and after each match (combines -A and -B).
- `--fmt jsonl` -- Emit flat JSONL records instead of text lines. Records use `kind=match|context|file`.

## EXAMPLES

Search for a string in a file:

    grep "TODO" /task_outputs/notes.md

Regex search recursively:

    grep -E "func\s+\w+" -r /knowledge_base

List files containing a pattern:

    grep -l "error" -r /task_outputs

Show context around matches:

    grep -C 2 "important" /knowledge_base/readme.md

Search stdin from a pipeline:

    cat /task_outputs/log.txt | grep "WARN"

Machine-readable JSONL output:

    grep --fmt jsonl "TODO" /task_outputs/notes.md

List matching files as JSONL:

    grep -l --fmt jsonl "error" -r /task_outputs

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Exit code is 1 when no matches are found.
- Only one path argument is accepted.
- Use `-r` when the path is a directory.
- Default text output remains pipe-friendly by design.
- `--fmt jsonl` is the explicit structured mode for agent parsing and downstream tooling.

## SEE ALSO

find, cat, sed

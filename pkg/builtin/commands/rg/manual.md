---
name: rg
synopsis: "rg [-F] [-i|-S] [-l] [-g GLOB]... [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH ...]"
category: search
---

# rg -- recursive text search with agent-friendly defaults

## SYNOPSIS

    rg [-F] [-i|-S] [-l] [-g GLOB]... [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH ...]
    COMMAND | rg [-F] [-i|-S] [-l] [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN
    rg --files [-g GLOB]... [PATH ...]

## DESCRIPTION

Search recursively through files using a scoped ripgrep-style builtin contract.
When no path is given, `rg` searches from the current virtual working directory.
When stdin is piped in and no path is given, `rg` searches stdin instead.

Default search output is one match per line using `path:line:text`. Context
lines use `path-line:text`, matching the existing `grep` text contract. The
`--files` mode lists candidate files without searching file contents.

This command is a builtin `simsh` contract, not a passthrough to a host
installed ripgrep binary. The supported flag surface is intentionally small and
focused on common agent search flows.

## FLAGS

- `-F` -- Use fixed-string matching instead of regex matching.
- `-i` -- Ignore case.
- `-S` -- Smart-case matching. Lowercase patterns ignore case; patterns with uppercase characters stay case-sensitive.
- `-l` -- List only file paths that contain a match.
- `-g GLOB`, `--glob GLOB` -- Restrict candidate files using shell-style glob filters.
- `--files` -- List searchable files instead of matching file contents.
- `-A N` -- Print N lines after each match.
- `-B N` -- Print N lines before each match.
- `-C N` -- Print N lines before and after each match.
- `--fmt jsonl` -- Emit flat JSONL records instead of text lines. Records use `kind=match|context|file`.

## EXAMPLES

Search recursively from the current working directory:

    cd /knowledge_base
    rg "TODO"

Restrict search to markdown files:

    rg -g "*.md" "hello" /knowledge_base

List only files that contain a match:

    rg -l "error" /task_outputs

List candidate files without searching contents:

    rg --files -g "*.json" /task_outputs

Search stdin from a pipeline:

    cat /task_outputs/log.txt | rg "WARN"

Machine-readable JSONL output:

    rg --fmt jsonl "TODO" /task_outputs

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Recursive search is the default; `-r` is accepted as a compatibility no-op.
- Line numbers are already part of the default text output; `-n` is accepted as a compatibility no-op.
- `--json` is accepted as a compatibility alias for `--fmt jsonl`, but `--fmt jsonl` is the canonical structured-mode contract.
- `-g` matches shell-style globs against file basenames and slash-normalized virtual paths.
- Negated glob patterns are not supported in the current implementation.
- Exit code is 1 when no matches are found.

## SEE ALSO

grep, find, json

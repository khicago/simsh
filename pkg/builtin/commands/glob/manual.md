---
name: glob
synopsis: "glob [--fmt jsonl] [--] PATTERN [PATH ...]"
category: navigation
---

# glob -- recursive path glob

## SYNOPSIS

    glob PATTERN
    glob PATTERN PATH...
    glob --fmt jsonl PATTERN PATH

## DESCRIPTION

List files whose relative path matches `PATTERN`. A pattern with no slash is
treated as recursive basename match, so `*.go` finds Go files at any depth
under the target. Use `**` when you want an explicit directory wildcard.

Default target is the session working directory.

## FLAGS

- `--fmt jsonl` -- Emit `{path,name,kind}` records instead of path-per-line text.
- `--` -- Stop option parsing so a pattern or relative path may begin with `-`.

## EXAMPLES

Find every Markdown file under knowledge:

    glob '**/*.md' /knowledge_base

Find Go files from the current directory:

    glob '*.go'

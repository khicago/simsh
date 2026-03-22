---
name: frontmatter
synopsis: inspect and print markdown YAML frontmatter across files
category: docs
---

# frontmatter -- inspect markdown frontmatter

## SYNOPSIS

```bash
frontmatter stat [-r] [--fmt compact|json|md] PATH...
frontmatter get [--raw] [--key KEY] PATH
frontmatter print [-r] [--key KEY] [-B N] [-A N] [-C N] [-n] PATH...
```

## DESCRIPTION

`frontmatter` provides a compact, agent-friendly way to inspect markdown frontmatter
across many files and print key sections with context.

Subcommands:

- `stat`: summarize whether each file has frontmatter, where it is, and key names.
- `get`: read the full frontmatter block (or one key value) from one file.
- `print`: print frontmatter (or one key block) with optional context lines.

## FLAGS

### Common

- `-r` -- recursively include files under directory targets.

### `stat`

- `--fmt compact|json|md` -- output format. Default `compact`.

### `get`

- `--key KEY` -- read one top-level key.
- `--raw` -- print raw YAML lines instead of parsed value text.

### `print`

- `--key KEY` -- print one top-level key block.
- `-B N` -- include `N` lines before the selected block.
- `-A N` -- include `N` lines after the selected block.
- `-C N` -- shorthand for `-B N -A N`.
- `-n` -- prefix output lines with line numbers.

## EXAMPLES

```bash
frontmatter stat docs -r
frontmatter stat --fmt json docs/must-guidebook.md docs/must-sop.md
frontmatter get --key title docs/must-guidebook.md
frontmatter print --key sop -C 2 docs/must-sop.md
```

## NOTES

- Frontmatter is detected only when the first line is `---` and a closing `---` exists.
- Paths may be absolute or relative to the current virtual working directory.
- `compact` output ends with a single legend line:
  - `# columns: has fm_lines key_count keys path`
- Default aliases include:
  - `fm` -> `frontmatter`
  - `ll` -> `ls -l`

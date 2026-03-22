---
name: json
synopsis: "json <stat|get> ..."
category: docs
---

# json -- inspect and extract JSON

## SYNOPSIS

```bash
json stat [-r] [--fmt compact|json|md] PATH...
json get [--path QUERY] [--raw] PATH
```

## DESCRIPTION

`json` is a small, agent-friendly JSON inspector. It is not `jq`.

Use it to:
- inspect JSON shape across files
- extract one subtree or scalar value without dumping a whole file

## FLAGS

### `stat`

- `-r` -- recursively include files under directory targets
- `--fmt compact|json|md` -- output format. Default `compact`

### `get`

- `--path QUERY` -- extract a subtree using a minimal path syntax such as `title`, `meta.author`, or `items[0].name`
- `--raw` -- emit compact JSON instead of pretty JSON for object/array results

## EXAMPLES

```bash
json stat /task_outputs/data.json
json stat -r --fmt json /workspace
json get --path items[0].name /task_outputs/data.json
json get --raw --path meta /task_outputs/data.json
```

## NOTES

- `json` intentionally does not implement jq-style filters or expression language.
- `json stat` reports validity, top-level kind, size, and key preview.
- `json get` returns:
  - plain text for scalar strings
  - JSON text for numbers, booleans, and null
  - pretty JSON for objects/arrays by default
  - compact JSON for objects/arrays with `--raw`

## SEE ALSO

frontmatter, cat, grep

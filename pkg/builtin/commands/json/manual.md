---
name: json
synopsis: "json <stat|get|keys|len> ..."
category: docs
---

# json -- inspect and extract JSON

## SYNOPSIS

```bash
json stat [-r] [--fmt compact|json|md] PATH...
json get [--path QUERY]... [--raw|--fmt json|--fmt jsonl] PATH
json keys [-r] [--path QUERY] [--fmt text|json|jsonl] PATH...
json len [-r] [--path QUERY] [--fmt text|json|jsonl] PATH...
```

## DESCRIPTION

`json` is a small, agent-friendly JSON inspector. It is not `jq`.

Use it to:
- inspect JSON shape across files
- extract one subtree or a small set of subtrees without dumping a whole file
- inspect object keys without re-reading full JSON into the model
- inspect container length without reconstructing it from raw text

## FLAGS

### `stat`

- `-r` -- recursively include files under directory targets
- `--fmt compact|json|md` -- output format. Default `compact`

### `get`

- `--path QUERY` -- extract one or more subtrees using a minimal path syntax such as `title`, `meta.author`, or `items[0].name`
- `--raw` -- emit compact JSON instead of pretty JSON for object/array results; for repeated `--path`, this applies to the combined object result
- `--fmt json` -- for repeated `--path`, emit one compact JSON object keyed by the exact query strings supplied on the command line
- `--fmt jsonl` -- for one or more `--path` values, emit one flat record per requested `--path` with fields `path`, `query`, and `value`

### `keys`

- `-r` -- recursively include files under directory targets
- `--path QUERY` -- resolve one subtree first, then list object keys from that subtree
- `--fmt text|json|jsonl` -- output format. Default `text`
  - succeeds only when the selected value is an object
  - invalid JSON, missing paths, arrays, numbers, booleans, and null become explicit per-file error rows instead of implicit coercions

### `len`

- `-r` -- recursively include files under directory targets
- `--path QUERY` -- resolve one subtree first, then report the length of that subtree
- `--fmt text|json|jsonl` -- output format. Default `text`
  - supports objects, arrays, and strings
  - invalid JSON, missing paths, numbers, booleans, and null become explicit per-file error rows instead of implicit coercions

## EXAMPLES

```bash
json stat /task_outputs/data.json
json stat -r --fmt json /workspace
json get --path items[0].name /task_outputs/data.json
json get --path meta.author --path items[1].name --fmt json /task_outputs/data.json
json get --raw --path meta /task_outputs/data.json
json keys --path meta /task_outputs/data.json
json len -r --path items /workspace
```

## NOTES

- `json` intentionally does not implement jq-style filters or expression language.
- `json stat` reports validity, top-level kind, size, and key preview.
- `json get` returns:
  - plain text for scalar strings
  - JSON text for numbers, booleans, and null
  - pretty JSON for objects/arrays by default
  - compact JSON for objects/arrays with `--raw`
- repeated `--path` turns `json get` into a small multi-extract tool, not a general query language:
  - default output is one pretty JSON object keyed by the exact query strings
  - `--raw` and `--fmt json` emit the same object in compact JSON form
  - `--fmt jsonl` emits one record per query in command-line order, even when only one `--path` is requested
- `json keys` is only for object-key inspection; it does not filter, map, transform, or coerce non-objects.
- `json len` is only for object/array/string length inspection; it does not aggregate across files beyond one row per file and does not coerce non-countable values.

## SEE ALSO

frontmatter, cat, grep

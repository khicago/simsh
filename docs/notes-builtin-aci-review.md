---
title: Builtin ACI Review
required: false
sop:
  - Read this doc when reviewing builtin command UX, manuals, or output contracts for agent use.
  - Update this doc when builtin default formats, command metadata, or manual-summary strategy changes.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Builtin ACI Review

## Context

`simsh` is not trying to be a smaller Bash clone. The kernel plan already says builtin commands should be reviewed through an agent-computer-interface lens, not a shell-completeness lens.

This review is subordinate to the current requirement baseline in `docs/notes-requirements.md`, especially:
- default outputs stay dual-readable;
- structured output is explicit and opt-in;
- pipeline-friendly commands should remain easy to compose;
- stronger query tools are preferred over repeated whole-file dumping.

That means builtin quality should be judged by whether an agent can:
- understand what a command expects without trial-and-error;
- parse the output without recovering unstated structure from prose;
- tell whether a mutation succeeded without always issuing follow-up reads;
- choose the right command without reading a long human-oriented manual first.

This does **not** mean every command should default to raw JSON or a machine-only representation. The higher bar is:
- readable by humans and agents at the same time;
- token-efficient by default;
- biased toward high signal-to-noise rather than toward maximal serialization.

## Review Rubric

This review uses four practical costs:
- parse cost: how much output decoding an agent must do;
- confirmation cost: how much extra work is needed to verify success;
- failure attribution cost: how hard it is to tell syntax/policy/path/semantic failures apart;
- token cost: how much formatting noise is spent per useful fact.

It also uses one design constraint:
- composition cost: whether a change preserves efficient shell-style piping and command chaining where that interaction model is already valuable.

## Current Status After K-006

This review is kept as the design input that drove the builtin ACI wave.
It is not a live inventory of missing flags.

Landed in the current tree:
- agent-native `edit` (unique snippet replace), `glob`, `view`, `dirname`, and `basename`;
- command-contract metadata on `BuiltinCommandDoc` and `man` / `man --list --fmt json`;
- dual-readable defaults for the worst token-waste commands, including `tree` defaulting to `outline` with `--fmt ascii|json`;
- `--fmt jsonl` on record streams such as `grep`, `rg`, and `find`;
- `--json` on object-style summaries such as `wc`, `env`, `type`, and mutation `--confirm/--json`;
- a constrained `json stat/get` inspector instead of a jq-style language.

If a later review needs a live inventory, read `simsh.md` and the command specs under `pkg/builtin/`, not the historical findings below as if they were still unimplemented.

## Historical Findings

These findings describe the pre-K-006 surface. Keep them for rationale, not as current status.

### 1. Structured output support is still sparse and inconsistent

The codebase already has a strong local pattern:
- `ls -l` supports `--fmt text|md|json` and keeps a fixed-column default text shape.
- `frontmatter stat` supports `compact|json|md`.

But most other builtin commands still expose only legacy text forms:
- `tree` only emits ASCII branches.
- `grep` only emits colon/dash-delimited text lines.
- `find` only emits plain path lines.
- `wc` emits unlabeled numeric tuples.
- `type`, `which`, `env`, and `man --list` only emit human text tables or sentences.

This means the builtin surface is directionally agent-native but not yet systematic.

### 2. Builtin descriptions are syntax-first, not contract-first

`BuiltinCommandDoc` only carries:
- `manual`
- `tips`
- `examples`
- `capabilities`

That is enough for CLI help, but not enough for agent-facing command selection. The missing contract fields are the ones agents branch on:
- whether stdin is required, optional, or ignored;
- whether operands are files, dirs, or generic paths;
- what the default output shape is;
- whether a machine format exists;
- what exit code `1` means for that command;
- whether successful mutation is silent or confirmatory.

`man` summary mode currently appends generic `Use-When` / `Avoid-When` hints, but those hints are not derived from an explicit command contract.

### 3. Path-heavy inspection commands still overfit to terminal tradition

The biggest signal-loss cases are:
- `tree`: ASCII branch art spends tokens on punctuation and indentation instead of path facts.
- `grep`: mixed `:` and `-` separators are familiar, but awkward for robust machine parsing, especially with context lines.
- `wc`: default unlabeled numbers require positional interpretation.
- `man --list`: aligned text columns optimize for human eyes, not stable field extraction.

For an agent runtime, "common Unix output" is not automatically the right default.

### 4. Mutation commands are too silent for text-only harnesses

`mkdir`, `touch`, `cp`, `mv`, `rm`, and `rmdir` all return empty stdout on success today.

That is acceptable for a traditional shell, but suboptimal for agent loops that only see stdout/stderr and are not directly consuming `ExecutionTrace`. The result is extra verification traffic:
- run mutation;
- run `ls` / `cat` / `find` to confirm;
- spend more tokens and time on confirmation than on the mutation itself.

This does not mean every mutation should become noisy by default. It does mean the builtin surface needs an explicit low-noise confirmation mode instead of relying only on follow-up reads.

### 5. The project already has a reusable design pattern worth standardizing

The best current examples are:
- `ls -l`: compact default text + explicit `--fmt` machine formats + stable legend.
- `frontmatter stat`: compact purpose-built default + `json|md` alternatives.

Those two commands should be treated as the reusable pattern for the next wave instead of letting every builtin reinvent its own output contract.

### 6. The builtin set is still weaker at structure-aware querying than at raw text dumping

There is a useful emerging pattern already:
- write structured content into the filesystem;
- query only the needed slice with a purpose-built inspector;
- avoid re-reading whole files with `cat` and then letting the model parse them from scratch.

`frontmatter` is the clearest example of the right direction. The first dedicated JSON follow-up should mirror that pattern with a constrained `json` inspector rather than a jq-style mini-language. The broader builtin set still leans too heavily on generic text tools (`cat`, `grep`, `sed -n`) for cases where structure-aware tools would save tokens and reduce parsing error.

## Command Review

| Command | Current State | Main ACI Problem | Recommended Direction | Priority |
| --- | --- | --- | --- | --- |
| `ls` | Strong baseline. Fixed columns in `-l`; `--fmt md|json` exists. | Default short listing still loses metadata unless caller remembers `-l`. | Keep current shape; treat `ls -l --fmt json` as the reference inspection contract. Consider surfacing `capabilities` in a concise text flag later. | Medium |
| `tree` | ASCII-only output. | High token cost and weak machine parseability. | Redesign around `--fmt ascii|outline|json`; make `outline` the agent-preferred format. Consider making non-ASCII-tree output the default for `simsh`, with `--fmt ascii` as opt-in compatibility. | High |
| `cd` / `pwd` | Semantics are now explicit and good. | Manuals should eventually expose state/change contract more directly. | Keep behavior. Minor manual contract cleanup only. | Low |
| `find` | Plain path-per-line default preserved. | Needed an explicit record mode without hurting pipe usage. | Keep path-per-line default. Add `--fmt jsonl` for planning flows and agent filtering. | Medium |
| `cat` | Raw text or numbered text; appropriate for content reads. | Large file reads can still be too unconstrained. | Keep raw-text default. Do not over-structure. Rely on `head`/`tail`/`sed -n` for slicing. | Low |
| `head` / `tail` | Good single-purpose text tools. | Minimal issue; mostly fine already. | Keep simple text shape. | Low |
| `grep` | Familiar text output with context support. | Match records were not explicit objects; context line shape was awkward for machine parsing. | Keep current text mode. Add `--fmt jsonl` with flat `match|context|file` records. | High |
| `rg` | Now implemented as a scoped ripgrep-style builtin with recursive-by-default search, cwd fallback, glob filters, and canonical `--fmt jsonl` output. | The risk is command-surface drift: caller familiarity can tempt the implementation toward host-binary passthrough or an ever-growing ripgrep clone. | Keep `rg` as the agent-oriented search front door, but keep the scope intentionally smaller than ripgrep: cover common inspect/search flows, reuse kernel path semantics, preserve project output taxonomy, and refuse host-binary passthrough or full CLI cloning. Keep `grep` as the simple text-first primitive and `find` as the path-discovery primitive rather than collapsing all three roles into one command. | High |
| `frontmatter` | Best-in-class builtin today. | Could become the template for other commands. | Preserve as reference pattern; no major redesign needed. | Low |
| `json` | Structure-aware inspector for JSON with `stat` and `get`. | Still too narrow for common agent queries, but at high risk of scope creep if expanded carelessly. | Keep the surface narrow and task-first: add `keys`, `len`, and small multi-path extraction to `get`; support batch-friendly outputs where that reduces token cost; make repeated `--path` output shape explicit and fixed; keep non-object/non-countable cases as explicit errors, not coercions; do not add filters, mutation, stdin, or jq-style expression semantics. | High |
| `echo` | Deterministic plain text. | None worth optimizing in kernel. | Keep as-is. | Low |
| `tee` | Useful bridge from stdin to file. | Needed explicit success feedback without breaking passthrough semantics. | Keep default passthrough. Add `--confirm` and `--json` as terminal-sink success summaries. | Medium |
| `sed` | Good split between print and in-place edit. | In-place mode needed an optional summary without touching print-mode pipes. | Keep `sed -n` text-first. Add `-i --json` for explicit mutation summaries. | Medium |
| `man` | Progressive disclosure is directionally right. | Summary/list output is prose-first; contract fields are implicit. | Add explicit command contract fields and let `man` render them. Also add `man --list --fmt json`. | High |
| `date` | Fine. Deterministic and low-surface. | None significant. | Keep as-is. | Low |
| `env` | Plain `KEY=VALUE` text works for humans. | List-like variables such as `PATH` needed better opt-in views. | Keep default text. Add `--json` and `--split` for one-key list-like inspection. | Medium |
| `mkdir` / `touch` | Silent success. | Agents needed explicit success feedback without changing shell defaults. | Keep silent default. Add `mkdir --confirm/--json`; add `touch --json`. | High |
| `cp` / `mv` | Silent success. | Needed explicit success confirmation and bytes summary. | Keep silent default. Add `--confirm` and `--json` summaries. | High |
| `rm` / `rmdir` | Silent success. | Deletes needed explicit success confirmation without weakening fail-fast semantics. | Keep silent default. Add `--confirm` and `--json` summaries. | High |
| `wc` | Functional but default output was weak. | Unlabeled numbers were positional and easy to misread. | Keep bare-number single-metric mode; switch multi-metric default to labeled compact text; add `--json`. | High |
| `sort` / `uniq` | Text transforms are fine for pipelines. | Little value in forcing structure by default. | Keep text default. Structured mode is optional, not urgent. | Low |
| `diff` | Unified diff is already a strong agent format for patch review. | No summary/stat mode for quick branching. | Keep unified diff default. Add optional `--fmt json` or `--stat` later if needed. | Medium |
| `which` | Path-per-line output is acceptable. | Needed explicit structured lookup summaries without changing default shell output. | Keep default path-per-line output. Add `--fmt json` with found/missing lookup entries. | Medium |
| `type` | Now better as fielded text. | Natural-language rows were too low-signal. | Use `name kind target` by default and add `--json` for structured resolution records. | Medium |

## Design Recommendations

### 1. Standardize one builtin output taxonomy

For commands that benefit from structured rendering, converge on:
- `--fmt text` for the default human-readable form where a multi-format family already exists;
- `--json` for commands that only need one explicit structured variant;
- `--fmt json` for commands that already have or should have a broader output family;
- `--fmt jsonl` for per-record streaming cases like `grep` and `find`;
- `--fmt md` only where human review tables are genuinely useful.

Do not invent one-off flags per command when one of these patterns is sufficient.

The split between `--json` and `--fmt jsonl` is deliberate:
- use `--json` when the command is fundamentally returning one object or summary;
- use `--fmt jsonl` when the command is fundamentally producing a stream of flat records;
- prefer the data-shape fit over superficial naming uniformity.

Compatibility aliases are acceptable only when they reduce high-frequency agent friction without changing the canonical contract.
That rule matters for commands such as `rg`:
- the project may accept a familiar compatibility flag when it maps cleanly onto an existing output taxonomy;
- the project should still document one canonical structured mode and one canonical output contract;
- compatibility aliases should stay parser-level only; they should not become the documented primary contract or force duplicate `BuiltinCommandDoc` surfaces for the same behavior;
- the project should not let compatibility flags become a back door for command-surface drift.

But the default should still optimize for dual-readability and token efficiency. In other words:
- do not default everything to JSON;
- do not preserve decorative terminal layouts just because they are traditional;
- prefer compact fielded text when it gives both humans and agents the same fact set at lower token cost.

### 2. Add explicit command-contract metadata

Extend builtin command metadata so `man` and other callers do not have to infer command behavior from prose alone.

The missing fields are roughly:
- stdin mode: `none | optional | required`
- operand kinds: `file | dir | path | command | text`
- default output kind: `raw | paths | table | diff | count | none`
- machine formats: `[]string`
- mutation behavior: `read_only | mutates`
- success stdout mode: `content | summary | silent`
- notable semantic exit codes

This metadata should become the SSOT for summary rendering, not a second prose layer.

### 3. Treat `ExecutionTrace` and builtin stdout as complementary, not interchangeable

API callers can consume `ExecutionTrace`, but text-only shells and harnesses still need stdout to be actionable.

The right design split is:
- stdout default stays concise;
- structured confirm modes exist where silence creates too much re-check traffic;
- manuals tell the caller when to rely on trace versus stdout.

### 4. Prefer fielded text over decorative text when text must stay default

Where JSON would be too heavy for the default, prefer:
- fixed columns with a legend;
- one record per line;
- labeled values instead of positional numeric tuples;
- no prose sentences when a stable row format would do.

This is why `tree` and `wc` deserve redesign sooner than `cat` or `sort`.

### 5. Preserve pipe composability where it already works well

The builtin surface should not sacrifice efficient command composition just to make structured output easier.

The case-by-case rule is:
- commands that are already good pipeline primitives should keep a pipe-friendly default and gain structured output through explicit flags;
- commands that are poor pipeline primitives anyway can spend their default output budget on higher-signal dual-readable presentation.

That is why:
- `cat`, `head`, `tail`, `sort`, `uniq`, and much of `grep` should remain text-first by default;
- `tree` can justify a stronger rethink of the default output shape;
- structured variants such as `--json` or `--fmt json` should be non-invasive additions, not forced replacements.

### 6. Add stronger structure-aware query tools instead of over-structuring every output

The runtime should get better at "read just the part I need", not only at "serialize everything more formally".

That means favoring tools like:
- `frontmatter` for Markdown frontmatter;
- the new builtin `json` for JSON shape inspection, subtree extraction, key inspection, and narrow container-length queries;
- future structure-aware inspectors for common agent file formats such as JSON, YAML, and tabular content;
- stronger local search tools that can return narrow, fielded results instead of large text blobs;
- one agent-friendly search front door (`rg`) that covers the most common multi-file text search flows without forcing callers to reconstruct them from `find` + `grep` every time;
- query-style subcommands that can extract keys, ranges, fields, or stats without forcing a full-file dump.

This is the better long-term complement to high-signal default text:
- humans keep readable defaults;
- agents keep token-efficient targeted access;
- structured files inside the virtual filesystem become more useful because the runtime helps query them directly.

### 7. Promote `ls -l` and `frontmatter stat` to reusable mechanisms

Those commands already express the right pattern:
- compact default text;
- explicit machine formats;
- SSOT-derived fields;
- stable legends and predictable columns.

The next wave should copy that pattern instead of inventing ad hoc output shapes command by command.

## Manual and Description Design

The manual layer should stop being only "what syntax exists" and start telling the agent "what contract it gets".

Each command manual should eventually declare:
- input contract
  - stdin: none / optional / required
  - accepted operand kinds
- default output contract
  - one record per line or raw text block
  - whether success is silent
- machine formats
  - `json`, `jsonl`, `md`, or none
- pipeline notes
  - whether the default output is intended to compose well in pipes
  - when the structured mode should be preferred instead
- semantic exit codes
  - especially for commands like `grep`, `diff`, and `which`

This is also the right long-term fix for `man --list`: it should be able to expose compact command cards instead of only a name + synopsis list.

For `rg`, the manual should keep three boundaries explicit up front:
- it is a builtin contract, not a passthrough to a host-installed ripgrep binary;
- it intentionally supports a focused compatibility subset rather than full ripgrep surface area;
- it reuses the project-wide structured-output taxonomy instead of inventing a private `rg`-only serialization model.

## Recommended Sequencing

### Phase 1: Metadata and manual contract layer
- Extend builtin metadata with explicit ACI contract fields.
- Upgrade `man` summary and `man --list` to use those fields.
- Align manuals around the same contract sections.

### Phase 2: High-value inspection command formats
- `tree`
- `grep`
- `find`
- `rg`
- `wc`
- `type`
- `env`

This phase should apply the case-by-case rule:
- change defaults more aggressively for commands with weak pipeline value;
- prefer additive structured flags for commands that agents are likely to compose.

### Phase 3: Structure-aware query tools
- treat `frontmatter` as the model for future structure parsers
- prioritize JSON inspection/query as the next default structured-data tool
- strengthen local search so agents can retrieve narrower result sets without broad `cat`/`grep` over-read
- add structure-first inspectors before broadening generic text dumping further
- prefer "extract the needed field/range/key" over "cat the whole file and let the model search"

### Phase 4: Mutation confirmation modes
- `mkdir`
- `touch`
- `cp`
- `mv`
- `rm`
- `rmdir`
- `tee`
- `sed -i`

### Phase 5: Nice-to-have follow-ups
- `diff --stat` or structured diff metadata
- optional richer listing overlays on top of `ls -l`

## Bottom Line

The strongest near-term optimization is not "add more commands". It is:
- make builtin contracts explicit;
- make default text outputs dual-readable and higher signal-to-noise;
- make high-volume inspection commands cheaper to parse;
- preserve shell composition where it is already a real strength;
- add stronger structure-aware query tools so agents can read only the part they need;
- make success/failure easier to branch on without follow-up probes.

For `simsh`, the best builtin surface is not the most Unix-like one. It is the one that lets an agent spend fewer tokens guessing and more tokens deciding.

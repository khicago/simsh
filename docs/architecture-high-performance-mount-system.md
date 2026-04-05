---
title: High-Performance Mount System
required: false
sop:
  - Read this doc before changing mount abstractions, adapter-backed filesystem projection behavior, or tool flows that can amplify mount access patterns.
  - Keep mount design explicit along semantic axes, capability contracts, and latency/consistency guarantees instead of relying on ad hoc fallback behavior.
  - When tool or mount changes would otherwise fall back to per-file RPC fanout, require explicit refusal or scope narrowing for `remote_high_latency` mounts instead of documenting the fallback and moving on.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# High-Performance Mount System

## Why This Doc Exists

`simsh` already treats filesystem projection as a first-class integration surface.
What it still needed was one explicit document for a harder question:

> when mounts stop being cheap local views and start sitting on top of DB, OS, RPC, search, or mixed persistence backends, what contract keeps `ls | cat | grep | rg | find | cp | mv | rm` usable instead of degenerating into implicit fanout?

This document is that contract.

It is not a generic filesystem essay.
It is the architecture and performance policy for mount-backed runtime surfaces inside `simsh`.

## Design Position

There are three common mistakes in mount-system design:

1. model mounts only by read/write
2. treat remote-backed files as transparent local files
3. leave high-fanout command behavior to fallback loops instead of capability contracts

All three fail once the mount backs onto RPC, DB, search, or hybrid storage.

The right model is:

- semantic axes describe what kind of truth a mount exposes
- capability contracts describe what operations the runtime may delegate efficiently
- stability and performance contracts describe when those capabilities remain trustworthy

The runtime should dispatch by those contracts.
It should not infer scalability from path shape or from whether a mount happens to respond to one narrow method.

## Scope

This doc covers:

- mount semantic models
- required capability interfaces for high-fanout CLI workloads
- consistency and freshness contracts
- latency and stability contracts
- runtime dispatch and fallback rules
- engineering guidance for adapter authors and tool authors

This doc does not define:

- product-specific business workflows
- one mandatory backend implementation
- a full distributed storage protocol

## Non-Goals

- pretending every mount is a cheap local directory
- forcing all mounts into one read-only projection model
- using writable mounts as a shortcut around explicit control-plane design
- allowing convenience fallbacks that silently expand into per-file RPC storms
- defining GUI or container-runtime semantics

## Core Principle

Mount design must be derived from **CLI pressure**, not only from backend shape.

The question is not “can this backend expose a file tree”.
The question is:

- what access pattern does the CLI create
- what fanout does that imply
- what capability must the mount expose so the runtime can execute that pattern without pathological cost

This is the only way to keep mount behavior aligned with an agent-native runtime.

## Semantic Axes

Mounts should be described along independent axes.
Do not collapse them into a single read-only vs writable taxonomy.

### 1. Truth Model

- `projection`
  - the mount exposes a derived or projected view over another source of truth
- `factual`
  - the mount exposes the primary operational file truth for the caller

This axis answers: **where does truth live?**

It does not answer whether writes are allowed.

### 2. Materialization Mode

- `snapshot`
  - a stable captured view
- `cached`
  - a locally materialized view that may diverge until refresh
- `live`
  - reads are expected to reflect current backend state directly

This axis answers: **how is visible data materialized?**

It does not answer whether writes persist immediately.

### 3. Write Semantics

- `read_only`
  - runtime mutations are not allowed through the mount
- `staged_write`
  - writes are accepted into an explicit staging layer
- `write_through`
  - writes are committed to the backing truth as part of the write path
- `write_back`
  - writes land locally first and propagate later under explicit policy

This axis answers: **what happens when the agent writes?**

### 4. Latency Class

- `local_fast`
- `local_heavy`
- `remote_moderate`
- `remote_high_latency`

This axis answers: **what fanout budget is safe?**

### 5. Consistency Contract

- `path_read_after_write`
- `list_after_write`
- `search_after_write`
- `refresh_required`

This axis answers: **what visibility guarantees does the caller actually get?**

## Why Read/Write Must Be Decoupled from Projection/Factual

A projection can be writable.
A factual mount can still be cached.

Examples:

- a projected resource tree may allow staged annotations without changing the original source immediately
- a factual workspace mount may use local write-back or snapshotting to protect a slower backend

So the right question is never:

“is this mount projection or writable?”

The right questions are:

- is the path projected truth or primary truth
- how is visible state materialized
- what write semantics apply
- what visibility guarantees follow

This decoupling is what keeps the model honest when backends get more complex.

## Mount Profiles

Every non-trivial mount should publish a profile with at least:

- truth model
- materialization mode
- write semantics
- latency class
- consistency guarantees
- supported CLI classes
- batch and search limits
- error and timeout semantics

Recommended shape:

```go
type MountProfile struct {
    TruthModel          string
    MaterializationMode string
    WriteSemantics      string
    LatencyClass        string
    Consistency         MountConsistency
    SupportedCLIClasses []string
    SLO                 MountSLO
}
```

This profile is not documentation fluff.
It is the dispatch contract for the runtime.

## Capability Contracts

The current `VirtualMount` contract is enough for narrow read-only projection behavior.
It is not enough to guarantee performance for high-fanout or writable workloads.

So the system should grow as a capability family, not as one ever-larger base interface.

### Base Identity and Metadata

Every mount still needs the current identity and metadata surface:

- mount point
- path existence
- path description
- path access and capabilities

This remains the minimum truth surface.

### Entry Listing

For any mount that wants to support `ls -l`, `tree`, or large directory traversal efficiently, name-only listing is not enough.

Required capability:

```go
type EntryLister interface {
    ListEntries(ctx context.Context, req ListEntriesRequest) (ListEntriesResult, error)
}
```

`ListEntries` should return:

- child name or display path
- kind
- access
- capabilities
- small stable metadata fields needed by `ls -l` / `tree`

This prevents the classic `ListChildren + N * DescribePath` fanout trap.

### Bulk Read

For any mount that expects multi-file reads, especially remote-backed mounts, repeated single-file reads are not acceptable as the primary path.

Required capability:

```go
type BulkReader interface {
    ReadMany(ctx context.Context, req ReadManyRequest) (ReadManyResult, error)
}
```

This is the capability that keeps:

- `cat a b c`
- batch `json stat`
- batch `frontmatter stat`
- multi-file inspection loops

from degenerating into linear RPC bursts.

### Path Enumeration

The existing runtime already has one notion of `search` that really means
**path discovery**:

- enumerate files under a scope
- support `CollectFilesUnder` / `ResolveSearchPaths`
- power directory-oriented traversal and planning

That meaning should stay separate from content search.

For any mount that wants to support `find`, recursive path discovery, or
directory-scoped planning efficiently, path enumeration must be an explicit
capability rather than an accidental by-product of `ListChildren`.

Required capability:

```go
type PathEnumerator interface {
    EnumeratePaths(ctx context.Context, req EnumeratePathsRequest) (EnumeratePathsResult, error)
}
```

### Content Search

For any mount that claims to support recursive content-search workloads, content search must not be implemented as:

- enumerate candidate files
- call `ReadRawContent` for each file
- scan client-side

Required capability:

```go
type ContentSearcher interface {
    SearchContent(ctx context.Context, req SearchRequest) (SearchResult, error)
}
```

This is the capability that should back:

- `grep -r`
- `rg`
- content-query follow-ups after path narrowing

Search should accept:

- path scope
- glob filters
- pattern
- fixed vs regex
- case mode
- context
- result limits

### Batch Mutation

For writable mounts, single-operation write hooks are not enough to guarantee correct semantics for composite tools.

Required capability:

```go
type Mutator interface {
    ApplyMutations(ctx context.Context, req MutationBatch) (MutationResult, error)
}
```

This is what keeps:

- `cp`
- `mv`
- `rm`
- `touch`
- `mkdir`
- `sed -i`
- `tee`

from becoming a multi-RPC consistency gamble.

### Refresh and Invalidations

For snapshot or cached mounts, refresh must be explicit.

Recommended capability:

```go
type Refresher interface {
    Refresh(ctx context.Context, req RefreshRequest) (RefreshResult, error)
}
```

Do not make ordinary reads double as hidden refresh triggers.

### Observability

For any non-trivial mount, performance and stability metadata should be queryable without scraping logs.

Recommended capability:

```go
type StatsProvider interface {
    Stats(ctx context.Context) (MountStats, error)
}
```

At minimum:

- request counts
- batch sizes
- refresh counts
- latency percentiles
- error counts
- stale or partial states

## CLI Pressure Matrix

This is the most practical part of the design.
The runtime should map each CLI class to the mount capabilities it requires.

| CLI / Pattern | Pressure Shape | Bad Fallback | Required Capability |
| --- | --- | --- | --- |
| `ls -l dir` | metadata fanout | `list + N*describe` | `EntryLister` |
| `tree dir` | recursive metadata fanout | recursive child enumeration + per-node describe | `EntryLister` with recursive support or equivalent batched tree listing |
| `find dir` | recursive path discovery | repeated list calls with deep path walks | `PathEnumerator` or equivalent batched recursive listing |
| `cat a b c` | multi-file body read | one RPC per file | `BulkReader` |
| `json/frontmatter stat dir` | batch inspect + parse | enumerate + repeated single reads | `BulkReader` |
| `grep -r` | enumerate + content scan | enumerate + per-file read + local scan | `ContentSearcher` |
| `rg` | recursive scoped search | same as above, but on a hotter path | `ContentSearcher` |
| `cp/mv/rm/touch/mkdir` | composite mutation | many independent writes/removes | `Mutator` |
| `sed -i` | read-modify-write | split read/write with no consistency contract | `Mutator` plus read-after-write guarantee |
| `ls | cat | grep` | chained amplification | each stage repeats remote traversal | typed dispatch plus no implicit fallback to per-file RPC |

## Runtime Dispatch Rules

The runtime should dispatch by capability and profile, not by optimism.

### Rule 1

If a mount advertises support for a CLI class, it must implement the corresponding capability directly.

### Rule 2

If a mount is `remote_high_latency`, missing a critical capability should cause:

- explicit refusal
- or explicit scope narrowing requirement

It should not silently fall back to fanout-heavy loops.

### Rule 3

If a mount is `local_fast`, narrower fallbacks may be acceptable, but they should still be visible and testable.

### Rule 4

The runtime should prefer:

- `ListEntries` over `ListChildren + DescribePath`
- `ReadMany` over repeated `ReadRawContent`
- `EnumeratePaths` over repeated recursive list walks
- `SearchContent` over enumerate-then-read
- `ApplyMutations` over ad hoc multi-operation sequences

### Rule 5

Writable mounts must declare visibility guarantees explicitly.
The runtime should not assume `search_after_write` just because `path_read_after_write` holds.

## Consistency Guarantees

The mount contract must distinguish:

- path read-after-write
- list-after-write
- search-after-write
- refresh-required semantics

Examples:

- `path_read_after_write`
  - write file, then immediate `cat path` sees the new content
- `list_after_write`
  - write file, then immediate `ls dir` includes the new entry
- `search_after_write`
  - write file, then immediate `rg pattern dir` includes the updated content

These are not interchangeable.
They need to be declared separately.

## Stability and Performance Contracts

Each mount that participates in non-trivial workloads should publish explicit operational limits.

At minimum:

- p50/p95/p99 for `list_entries`
- p50/p95/p99 for `read_many`
- p50/p95/p99 for `search`
- p50/p95/p99 for `apply_mutations`
- max batch count
- max batch bytes
- max search scope
- timeout semantics
- partial-result semantics
- retryability classification

This is how the runtime and the integrator decide whether a mount can safely participate in a CLI class.

## Fallback Policy

Not all fallback is forbidden.
But fallback must be explicit and bounded.

Allowed examples:

- local-fast mount using `ReadRawContent` for single-file `cat`
- small test mount using simple recursive enumeration for `find`

Disallowed examples:

- remote-high-latency mount implementing `rg` through file-by-file reads
- writable factual mount implementing `mv` as best-effort copy-then-delete without explicit mutation guarantees
- large mount implementing `ls -l` by listing names and then describing each child separately over RPC

## Interaction With Existing Projection Architecture

This document does not replace the existing projection docs.
It refines them.

Current projection architecture remains valid:

- projection freshness should be explicit
- refresh should be explicit
- managed views should stay read-only unless a control-plane contract says otherwise

What this document adds is the performance and dispatch side:

- how capability shape must change under fanout pressure
- how writable or factual mounts fit without collapsing back into opaque remote filesystems
- how CLI combinations turn into concrete backend stress

## Contract Boundary and Migration Rule

The old `VirtualMount` shape bundled too many responsibilities:

- path description
- direct children listing
- recursive path discovery
- content reads
- content-search preparation

That bundled surface is exactly what makes high-fanout behavior ambiguous.

So the end-state contract is:

- `VirtualMount` owns only identity, profile, existence, and single-path metadata truth
- capability interfaces own batched or pressure-sensitive behavior

In other words:

- `ListChildren`
- `IsDirPath`
- `ReadRawContent`
- `CollectFilesUnder`
- `ResolveSearchPaths`
- `DescribePath`

must stop being treated as the canonical mount contract once the capability family lands.

If a temporary bridge is used during refactor, it is implementation-local only.
It must not become a second public SSOT.

## Guidance for Integrators

When choosing a mount model, start from workload, not backend pride.

If your workload is mostly:

- browse + inspect
  - optimize `EntryLister`
- batch content inspection
  - optimize `ReadMany`
- recursive search
  - optimize `Searcher`
- factual workspace writes
  - optimize `Mutator` and consistency guarantees

If you cannot provide the required capability efficiently, say so in the profile and let the runtime reject or narrow the workload.
That is better than pretending support and degrading at runtime.

## Guidance for Tool Authors

Tool and builtin changes are not mount-neutral.

When changing:

- `ls`
- `tree`
- `find`
- `grep`
- `rg`
- `cat`
- `json`
- `frontmatter`
- `cp`
- `mv`
- `rm`
- `touch`
- `mkdir`
- any new tool that multiplies path fanout

you must evaluate:

- what mount capability it requires
- what fallback path it triggers today
- whether that fallback is acceptable for remote or high-latency mounts
- whether visibility and consistency guarantees still hold

Tool ergonomics without mount-aware dispatch is not a harmless refactor.
It changes backend stress shape.

## Rollout Strategy

1. make the capability taxonomy explicit in docs first
2. expose profile and capability metadata in mount-facing contracts
3. update builtin dispatch to prefer strong capabilities over fallback loops
4. make high-latency fallback failures explicit
5. add benchmark coverage for mount-heavy CLI combinations

## Bottom Line

The high-performance mount problem is not “how do we cache mounts”.

The real problem is:

- how truth model, materialization mode, write semantics, latency class, and consistency guarantees combine
- how CLI composition amplifies backend pressure
- how the runtime prevents innocent-looking commands from turning into pathological fanout

The architecture should therefore optimize for:

- explicit semantic axes
- explicit capability contracts
- explicit SLO and consistency guarantees
- explicit dispatch and refusal rules

That is how `simsh` keeps mount-backed environments usable once the backend stops being a cheap local directory.

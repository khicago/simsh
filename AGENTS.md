<!-- BAGAKIT:LIVEDOCS:START -->
This is a managed block. Do not edit content between START/END tags directly; it may be overwritten by re-running the Bagakit apply script. Edit the Bagakit templates/scripts instead.

Tooling:
- Resolve the installed skill dir as: `export BAGAKIT_LIVING_DOCS_SKILL_DIR="${BAGAKIT_LIVING_DOCS_SKILL_DIR:-${BAGAKIT_HOME:-$HOME/.claude}/skills/bagakit-living-docs}"`

System-level requirements (must):
- Read all system docs: `docs/must-*.md` (especially `docs/must-guidebook.md`, `docs/must-docs-taxonomy.md`, `docs/must-memory.md`).
- Follow `docs/must-sop.md` (generated from docs frontmatter). If SOP sources change, regenerate via `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_generate_sop.sh" .`.
- Before answering questions about prior work/decisions/todos/preferences: search `docs/.bagakit/memory/**/*.md`/`docs/.bagakit/inbox/**/*.md`/`docs/**/*.md` via `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_memory.sh" search '<query>' --root .`, then use `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_memory.sh" get <path> --root . --from <line> --lines <n>` to quote only needed lines.
- Before creating new docs or memory entries, search first; prefer updating/merging an existing canonical entry over creating near-duplicates.

Optional mechanisms (adopt per the target project's own norms):
- Reusable-items governance/catalogs (e.g. `docs/norms-maintaining-reusable-items.md`, `docs/notes-reusable-items-*.md`): use if it helps your project converge on standards; if you decide not to use it, delete/ignore the docs and record the decision.
- Response directives (`directives:` in doc frontmatter): only apply when your project defines and uses them.

Workflow helpers:
If you capture a new durable memory during work, write it to `docs/.bagakit/inbox/` using `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_inbox.sh" new <kind> <topic> --root . --title '<title>'`, then promote after review with `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_inbox.sh" promote docs/.bagakit/inbox/<file>.md --root .` (use `--merge` if the curated target already exists).
If Bagakit docs/memory are missing, bootstrap/update the project by running: `bash "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/apply-living-docs.sh" .` (use `--force` only when you intend to overwrite templates).
If you want to check whether the installed skill is up to date with a remote branch, run: `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_update.sh" status` (optionally pass `--repo <git_url>`).
When you change docs/memory rules or tooling, run `sh "$BAGAKIT_LIVING_DOCS_SKILL_DIR/scripts/bagakit_living_docs_doctor.sh" .` and either apply suggested fixes or record a decision in inbox.

Read `docs/must-guidebook.md` before working; follow `docs/must-sop.md`; use recall (search -> get) for prior decisions.
Read `docs/must-guidebook.md`; in the final response, include the `[[BAGAKIT]]` footer block (LivingDoc + apply directives when applicable).

At the end of every response, include:
- `[[BAGAKIT]]`
- `- LivingDoc: <short note about which system doc rule was followed>`
- `  - (<DIRECTIVE>) <optional directive output when applicable>`
<!-- BAGAKIT:LIVEDOCS:END -->

<!-- BAGAKIT:LONGRUN:START -->
This is a managed block. Do not edit content between START/END tags directly; it may be overwritten by re-running the Bagakit apply script. Edit the Bagakit templates/scripts instead.

Role:
- `long-run` is the execution-loop driver over upstream execution systems.
- Upstream systems keep source-of-truth state; `long-run` normalizes one actionable item at a time.

Tooling:
- Resolve the installed skill dir as: `export BAGAKIT_LONG_RUN_SKILL_DIR="${BAGAKIT_LONG_RUN_SKILL_DIR:-${BAGAKIT_HOME:-$HOME/.claude}/skills/bagakit-long-run}"`

Loop Protocol (every round):
1. Run `sh .bagakit/long-run/init.sh`.
2. If detect is not ready, run `.bagakit/long-run/detect_prompt.md`, update `.bagakit/long-run/bk-execution-table.json`, then re-run init.
3. Run initializer pass with `.bagakit/long-run/initial_prompt.md` and produce a single-item handoff.
4. Run coding pass with `.bagakit/long-run/coding_prompt.md`, execute exactly one item, and update status/check evidence.
5. Re-run `sh .bagakit/long-run/init.sh` to actively pick the next actionable item and continue.

Response Driver (every long-run pass):
- End detect/initializer/coding responses with the project footer block `[[BAGAKIT]]`.
- Add long-run progress as a peer footer line (same level as `- LivingDoc: ...`):
  - `- LongRun: Item=<id|detect>; Status=<ready|todo|in_progress|done|blocked>; Evidence=<validation/tests>; Next=<exact command>`

Detect Rules:
- Detect is agent-driven and rule-based; do not hardcode project-specific names in scripts.
- Map upstream systems to adapters (`bagakit-ft`, `openspec`, `manual`) in `.bagakit/long-run/bk-execution-table.json`.
- Every execution row must include: why-now, binary acceptance criteria, files to touch, commands, and risk/rollback notes.

Validation:
- `bash "$BAGAKIT_LONG_RUN_SKILL_DIR/scripts/validate-long-run.sh" .`
- `bash "$BAGAKIT_LONG_RUN_SKILL_DIR/scripts/bagakit_long_run_doctor.sh" .`
<!-- BAGAKIT:LONGRUN:END -->


---
title: Repo-local skills live under .codex/skills with stable alias symlinks
kind: decision
status: inbox
tags:
  - decision
sources:
  - .codex/skills/qihan-golang-testing-v0/SKILL.md
  - .codex/skills/qihan-golang-testing-v0/manifest.yaml
  - .codex/skills/qihan-golang-testing-v0/tools/validate.sh
created: 2026-03-22T20:27:07Z
---

## Candidate
- Context: while creating the repo-local `qihan-golang-testing` skill, the repo needed a stable local installation point that is versionable and easy to reference from future tasks.
- Decision:
  - keep versioned skill directories under `.codex/skills/<skill-name>-vN/`
  - expose the stable command/entry name as a symlink `.codex/skills/<skill-name> -> <skill-name>-vN`
  - keep the actual skill files in-repo rather than only under `~/.agents/skills`, so the skill can be versioned and reviewed with the codebase
- Why:
  - repo-local skills are inspectable and commit-friendly
  - versioned directories preserve creator/evolution history
  - stable alias symlinks keep invocation names clean
- Example:
  - `.codex/skills/qihan-golang-testing-v0/`
  - `.codex/skills/qihan-golang-testing -> qihan-golang-testing-v0`

## Promote To
- `docs/.bagakit/memory/decision-repo-local-skills.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

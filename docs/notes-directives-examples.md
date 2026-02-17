---
title: Response Directives (Examples)
required: false
sop:
  - Read this doc when you want to introduce or change response directives (`directives:` in doc frontmatter).
  - Keep directive usage practical: only add directives that reduce mistakes or improve debuggability.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Response Directives (Examples)

Directives are optional, project-defined "driver statements" that guide how an agent writes its final response when a condition applies.

They are defined in doc frontmatter under `directives:` and are surfaced in `docs/must-sop.md` so they are discoverable.

This doc provides example directives and patterns. Adopt/modify as needed for your project.

## Example: DEBUG Attempt Tracking
Motivation: avoid repeating the same unproductive debugging loop without learning.

Suggested directive (put in a relevant debugging/runbook doc frontmatter):
```yaml
directives:
  - DEBUG: When debugging, include a final-response attempt summary using: `<problem> (try-<n>): <one-line approach>`.
```

Optional extension pattern (project-defined):
- Keep a running counter per hypothesis/approach (try-1/2/3...).
- If the same approach exceeds a threshold (e.g. 3 tries), explicitly mark it as likely-not-working and switch strategy:
  - add instrumentation/logs
  - validate assumptions
  - request missing repro info
  - record a gotcha candidate into `docs/.bagakit/inbox/`

## Example: RISK Callouts for High-Stakes Changes
Motivation: make risk visible when touching sensitive areas.

```yaml
directives:
  - RISK: When touching auth/payments/data migration, include a `Risks` section with: impact, rollback plan, and how it was validated.
```

## Example: CHANGELOG Notes When Updating Governance
Motivation: reduce silent policy drift.

```yaml
directives:
  - GOV: When changing docs/memory rules or tooling, include: what changed, why, and what to do next.
```

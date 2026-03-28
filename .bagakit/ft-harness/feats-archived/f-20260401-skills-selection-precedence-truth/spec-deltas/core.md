# Spec Delta: core

## ADDED Requirements

### Requirement: Skills selection truth stays adapter-local and explainable
The adapter-facing `/skills` projection SHALL expose selection as a derived truth surface when the adapter declares an explicit competition scope.

#### Scenario: Explicit selection scope yields one deterministic winner
- **WHEN** multiple eligible skills share the same explicit selection scope
- **THEN** the adapter selects exactly one winner using documented precedence inputs and deterministic tie-breaks
- **AND** non-selected skills remain visible with an explanation for why they lost

#### Scenario: Unscoped skills do not invent competition
- **WHEN** a skill has no explicit selection scope
- **THEN** the adapter does not infer competition from path layout alone
- **AND** any exposed `selected` state must still be explainable by explicit adapter metadata or singleton behavior

#### Scenario: Ineligible skills cannot be silently selected
- **WHEN** a skill is marked ineligible
- **THEN** it is never surfaced as selected
- **AND** the reason remains visible in skill metadata

## MODIFIED Requirements

### Requirement: `/skills` metadata includes selection provenance
The `/skills` index and mirrored `/memory/projections.json` view SHALL include explicit selection provenance when selection state is exposed, in addition to compatibility fields such as `selected`.

## REMOVED Requirements

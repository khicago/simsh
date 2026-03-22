# Spec Delta: core

## ADDED Requirements

### Requirement: Execution trace records actionable side effects
The system SHALL return direct side-effect and denial information in a structured execution trace suitable for planner and adapter consumption.

#### Scenario: File-mutating command emits trace categories
- **WHEN** a command reads, writes, edits, removes, or is denied on virtual paths
- **THEN** the execution trace reports the attempted and successful path effects in stable categories

## MODIFIED Requirements

## REMOVED Requirements

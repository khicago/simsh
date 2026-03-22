# Spec Delta: core

## ADDED Requirements

### Requirement: New runtime contracts require adapter-backed validation
The system SHALL validate provisional session, trace, and adapter contracts against at least one adapter-backed end-to-end workload before treating them as stable.

#### Scenario: Reference workload validates seam behavior
- **WHEN** the reference adapter runs a representative end-to-end workflow
- **THEN** it demonstrates projection, trace consumption, session resume, and optional memory lifecycle behavior together

## MODIFIED Requirements

## REMOVED Requirements

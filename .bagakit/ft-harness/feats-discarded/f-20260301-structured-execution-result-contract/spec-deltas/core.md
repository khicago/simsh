# Spec Delta: core

## ADDED Requirements

### Requirement: Structured execution result is the core source of truth
The system SHALL produce a structured execution result contract that higher-level surfaces can render or transport without losing machine-consumable fields.

#### Scenario: HTTP and CLI share one execution result model
- **WHEN** a command finishes successfully or unsuccessfully
- **THEN** runtime surfaces derive their presentation from the same structured result instead of independent text-only code paths

## MODIFIED Requirements

## REMOVED Requirements

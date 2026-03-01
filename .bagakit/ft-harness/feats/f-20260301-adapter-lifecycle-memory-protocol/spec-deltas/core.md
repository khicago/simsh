# Spec Delta: core

## ADDED Requirements

### Requirement: Adapters can participate in session lifecycle explicitly
The system SHALL expose adapter-facing lifecycle hooks and optional memory protocol hooks without hard-coding one business domain into core runtime packages.

#### Scenario: Adapter-managed memory checkpoint
- **WHEN** a session checkpoint or close event occurs for an adapter that implements optional memory hooks
- **THEN** the adapter receives an explicit chance to hydrate, observe, checkpoint, or flush managed state

## MODIFIED Requirements

## REMOVED Requirements

# Spec Delta: core

## ADDED Requirements

### Requirement: Session lifecycle is a first-class runtime contract
The system SHALL support generic session create/resume/checkpoint/close flows and session-scoped policy ceilings without breaking one-shot execution.

#### Scenario: Resumeable session execution
- **WHEN** a caller resumes an existing session and executes another command within its policy ceiling
- **THEN** runtime state is restored intentionally and the effective policy cannot exceed the session ceiling

## MODIFIED Requirements

## REMOVED Requirements

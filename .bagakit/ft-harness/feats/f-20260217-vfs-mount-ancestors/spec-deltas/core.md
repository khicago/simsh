# Spec Delta: core

## ADDED Requirements

### Requirement: Synthetic mount parent directories are visible
The system SHALL expose synthetic parent directories for active virtual mount points.

#### Scenario: List mount parent directory
- **WHEN** `/sys/bin` mount exists and is active
- **THEN** `ls /sys` succeeds and includes `/sys/bin` in its output

### Requirement: Mount-backed and synthetic mount-parent paths are immutable
The system SHALL reject filesystem mutations targeting:
- mount-backed paths (any path under a mount point)
- synthetic mount-parent paths (directories that exist solely to parent active mount points)

#### Scenario: Reject mkdir on mount parent
- **WHEN** executing `mkdir /sys` under a write-allowed policy
- **THEN** the command fails with an "unsupported" error

#### Scenario: Reject remove/copy/move on mount-backed paths without partial writes
- **WHEN** executing `rm /sys/bin/ls`
- **THEN** the command fails with an "unsupported" error
- **WHEN** executing `cp /workspace/readme.md /sys/bin/copied`
- **THEN** the command fails with an "unsupported" error
- **WHEN** executing `mv /sys/bin/ls /workspace/moved`
- **THEN** the command fails with an "unsupported" error AND `/workspace/moved` is NOT created

## MODIFIED Requirements

## REMOVED Requirements

## Planned (Not Implemented Yet)

### Requirement: Path access metadata is explicit and machine-readable
The system SHOULD expose a SSOT `access=ro|rw` and `capabilities[]` in path metadata and surface it in:
- `ls -l` (short columns + legend)
- `ls -l --fmt md|json`
- `/v1/execute` response meta (opt-in)

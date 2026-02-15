package contract

import (
	"context"
	"time"
)

const (
	AuditPhaseCommandStart = "command_start"
	AuditPhaseCommandEnd   = "command_end"
	AuditPhaseCommandError = "command_error"
)

// AuditEvent is emitted around command execution for observability.
type AuditEvent struct {
	Time     time.Time
	Phase    string
	Command  string
	Args     []string
	ExitCode int
	Message  string
}

type AuditSink interface {
	Emit(ctx context.Context, event AuditEvent)
}

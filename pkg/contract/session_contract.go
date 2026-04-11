package contract

import (
	"encoding/json"
	"strings"
	"time"
)

// Session captures runtime session state across execute calls, including
// resumable state and transient live-execution observation fields.
type Session struct {
	SessionID       string                 `json:"session_id"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Profile         CompatibilityProfile   `json:"profile"`
	PolicyCeiling   ExecutionPolicy        `json:"policy_ceiling"`
	State           SessionState           `json:"state"`
	ActiveExecution *SessionExecutionState `json:"active_execution,omitempty"`
}

// SessionState keeps only runtime-scoped continuation data.
type SessionState struct {
	CommandAliases map[string][]string        `json:"command_aliases,omitempty"`
	EnvVars        map[string]string          `json:"env_vars,omitempty"`
	RCFiles        []string                   `json:"rc_files,omitempty"`
	WorkingDir     string                     `json:"working_dir,omitempty"`
	Opaque         map[string]json.RawMessage `json:"opaque,omitempty"`
}

type SessionExecutionStatus string

const (
	SessionExecutionStatusRunning   SessionExecutionStatus = "running"
	SessionExecutionStatusCanceling SessionExecutionStatus = "canceling"
)

type SessionExecutionState struct {
	ExecutionID     string                 `json:"execution_id"`
	CommandLine     string                 `json:"command_line"`
	StartedAt       time.Time              `json:"started_at"`
	Status          SessionExecutionStatus `json:"status"`
	StatusUpdatedAt time.Time              `json:"status_updated_at"`
}

func (s Session) Clone() Session {
	s.State = s.State.Clone()
	s.PolicyCeiling = s.PolicyCeiling.Clone()
	if s.ActiveExecution != nil {
		cloned := *s.ActiveExecution
		s.ActiveExecution = &cloned
	}
	return s
}

func (s SessionState) Clone() SessionState {
	out := SessionState{
		CommandAliases: NormalizeCommandAliases(s.CommandAliases),
		EnvVars:        NormalizeEnvVars(s.EnvVars),
		RCFiles:        NormalizeRCFiles(s.RCFiles),
		WorkingDir:     strings.TrimSpace(s.WorkingDir),
	}
	if len(s.Opaque) > 0 {
		out.Opaque = make(map[string]json.RawMessage, len(s.Opaque))
		for key, value := range s.Opaque {
			out.Opaque[key] = append(json.RawMessage(nil), value...)
		}
	}
	return out
}

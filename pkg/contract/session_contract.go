package contract

import (
	"encoding/json"
	"strings"
	"time"
)

// Session captures resumable runtime state across execute calls.
type Session struct {
	SessionID     string               `json:"session_id"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Profile       CompatibilityProfile `json:"profile"`
	PolicyCeiling ExecutionPolicy      `json:"policy_ceiling"`
	State         SessionState         `json:"state"`
}

// SessionState keeps only runtime-scoped continuation data.
type SessionState struct {
	CommandAliases map[string][]string        `json:"command_aliases,omitempty"`
	EnvVars        map[string]string          `json:"env_vars,omitempty"`
	RCFiles        []string                   `json:"rc_files,omitempty"`
	WorkingDir     string                     `json:"working_dir,omitempty"`
	Opaque         map[string]json.RawMessage `json:"opaque,omitempty"`
}

func (s Session) Clone() Session {
	s.State = s.State.Clone()
	s.PolicyCeiling = s.PolicyCeiling.Clone()
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

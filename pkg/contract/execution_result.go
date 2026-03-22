package contract

import (
	"strings"
	"time"
)

// ExecutionResult is the structured source of truth for one execution.
type ExecutionResult struct {
	ExecutionID string         `json:"execution_id"`
	SessionID   string         `json:"session_id,omitempty"`
	ExitCode    int            `json:"exit_code"`
	Stdout      string         `json:"stdout"`
	Stderr      string         `json:"stderr"`
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  time.Time      `json:"finished_at"`
	DurationMS  int64          `json:"duration_ms"`
	Trace       ExecutionTrace `json:"trace"`
}

// ExecutionTrace is the machine-consumable execution summary attached to results.
type ExecutionTrace struct {
	CommandLine         string               `json:"command_line,omitempty"`
	Command             string               `json:"command,omitempty"`
	Argv                []string             `json:"argv,omitempty"`
	Pipeline            []ExecutionTraceStep `json:"pipeline,omitempty"`
	EffectiveProfile    CompatibilityProfile `json:"effective_profile,omitempty"`
	EffectivePolicy     ExecutionPolicy      `json:"effective_policy"`
	TimedOut            bool                 `json:"timed_out,omitempty"`
	Canceled            bool                 `json:"canceled,omitempty"`
	RequestedPaths      []string             `json:"requested_paths,omitempty"`
	ReadPaths           []string             `json:"read_paths,omitempty"`
	WrittenPaths        []string             `json:"written_paths,omitempty"`
	AppendedPaths       []string             `json:"appended_paths,omitempty"`
	EditedPaths         []string             `json:"edited_paths,omitempty"`
	CreatedDirs         []string             `json:"created_dirs,omitempty"`
	RemovedPaths        []string             `json:"removed_paths,omitempty"`
	DeniedPaths         []string             `json:"denied_paths,omitempty"`
	BytesRead           int                  `json:"bytes_read,omitempty"`
	BytesWritten        int                  `json:"bytes_written,omitempty"`
	ExternalStdoutBytes int                  `json:"external_stdout_bytes,omitempty"`
	ExternalStderrBytes int                  `json:"external_stderr_bytes,omitempty"`
	OutputTruncated     bool                 `json:"output_truncated,omitempty"`
}

type ExecutionTraceStep struct {
	Command string   `json:"command"`
	Argv    []string `json:"argv,omitempty"`
}

func (r ExecutionResult) FlattenOutput() string {
	switch {
	case r.Stdout == "":
		return r.Stderr
	case r.Stderr == "":
		return r.Stdout
	case strings.HasSuffix(r.Stdout, "\n"):
		return r.Stdout + r.Stderr
	default:
		return r.Stdout + "\n" + r.Stderr
	}
}

func (r ExecutionResult) WithSessionID(sessionID string) ExecutionResult {
	r.SessionID = strings.TrimSpace(sessionID)
	return r
}

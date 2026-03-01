package contract

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrPolicyCeilingExceeded = errors.New("requested policy exceeds session ceiling")

type WriteMode string

const (
	WriteModeDisabled     WriteMode = "disabled"
	WriteModeReadOnly     WriteMode = "read-only"
	WriteModeWriteLimited WriteMode = "write-limited"
	WriteModeFull         WriteMode = "full"
)

// ExecutionPolicy controls runtime safety bounds.
type ExecutionPolicy struct {
	WriteMode        WriteMode
	MaxWriteBytes    int
	MaxPipelineDepth int
	MaxOutputBytes   int
	Timeout          time.Duration
}

func DefaultPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		WriteMode:        WriteModeReadOnly,
		MaxWriteBytes:    1 << 20,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          15 * time.Second,
	}
}

func (p ExecutionPolicy) Clone() ExecutionPolicy {
	return p
}

func PolicyPreset(name string) (ExecutionPolicy, error) {
	policy := DefaultPolicy()
	switch strings.TrimSpace(name) {
	case "", string(WriteModeReadOnly):
		policy.WriteMode = WriteModeReadOnly
	case string(WriteModeDisabled):
		policy.WriteMode = WriteModeDisabled
	case string(WriteModeWriteLimited):
		policy.WriteMode = WriteModeWriteLimited
	case string(WriteModeFull):
		policy.WriteMode = WriteModeFull
	default:
		return ExecutionPolicy{}, fmt.Errorf("unsupported policy %q", name)
	}
	return policy, nil
}

func (p ExecutionPolicy) AllowWrite() bool {
	switch p.WriteMode {
	case WriteModeFull, WriteModeWriteLimited:
		return true
	default:
		return false
	}
}

func (p ExecutionPolicy) WithInheritedUnset(limit ExecutionPolicy) ExecutionPolicy {
	out := p
	if out.WriteMode == "" {
		out.WriteMode = limit.WriteMode
	}
	if out.MaxWriteBytes == 0 {
		out.MaxWriteBytes = limit.MaxWriteBytes
	}
	if out.MaxPipelineDepth == 0 {
		out.MaxPipelineDepth = limit.MaxPipelineDepth
	}
	if out.MaxOutputBytes == 0 {
		out.MaxOutputBytes = limit.MaxOutputBytes
	}
	if out.Timeout == 0 {
		out.Timeout = limit.Timeout
	}
	return out
}

func PolicyWithinCeiling(requested ExecutionPolicy, ceiling ExecutionPolicy) error {
	if writeModeRank(requested.WriteMode) > writeModeRank(ceiling.WriteMode) {
		return fmt.Errorf("%w: requested write mode %q exceeds session ceiling %q", ErrPolicyCeilingExceeded, requested.WriteMode, ceiling.WriteMode)
	}
	if ceiling.WriteMode == WriteModeWriteLimited && requested.WriteMode == WriteModeWriteLimited &&
		ceiling.MaxWriteBytes > 0 && requested.MaxWriteBytes > ceiling.MaxWriteBytes {
		return fmt.Errorf("%w: requested max_write_bytes %d exceeds session ceiling %d", ErrPolicyCeilingExceeded, requested.MaxWriteBytes, ceiling.MaxWriteBytes)
	}
	if ceiling.MaxPipelineDepth > 0 && requested.MaxPipelineDepth > ceiling.MaxPipelineDepth {
		return fmt.Errorf("%w: requested max_pipeline_depth %d exceeds session ceiling %d", ErrPolicyCeilingExceeded, requested.MaxPipelineDepth, ceiling.MaxPipelineDepth)
	}
	if ceiling.MaxOutputBytes > 0 && requested.MaxOutputBytes > ceiling.MaxOutputBytes {
		return fmt.Errorf("%w: requested max_output_bytes %d exceeds session ceiling %d", ErrPolicyCeilingExceeded, requested.MaxOutputBytes, ceiling.MaxOutputBytes)
	}
	if ceiling.Timeout > 0 && requested.Timeout > ceiling.Timeout {
		return fmt.Errorf("%w: requested timeout %s exceeds session ceiling %s", ErrPolicyCeilingExceeded, requested.Timeout, ceiling.Timeout)
	}
	return nil
}

func EffectivePolicyWithinCeiling(requested ExecutionPolicy, ceiling ExecutionPolicy) (ExecutionPolicy, error) {
	if ceiling.WriteMode == "" {
		ceiling = DefaultPolicy()
	}
	effective := requested.WithInheritedUnset(ceiling)
	if effective.WriteMode == "" {
		effective.WriteMode = ceiling.WriteMode
	}
	if err := PolicyWithinCeiling(effective, ceiling); err != nil {
		return ExecutionPolicy{}, err
	}
	return effective, nil
}

func writeModeRank(mode WriteMode) int {
	switch mode {
	case WriteModeDisabled:
		return 0
	case WriteModeReadOnly:
		return 1
	case WriteModeWriteLimited:
		return 2
	case WriteModeFull:
		return 3
	default:
		return 1
	}
}

// CheckWriteSize returns an error if writing contentLen bytes is not allowed by policy.
func (p ExecutionPolicy) CheckWriteSize(contentLen int) error {
	if !p.AllowWrite() {
		return ErrUnsupported
	}
	if p.WriteMode == WriteModeWriteLimited && p.MaxWriteBytes > 0 && contentLen > p.MaxWriteBytes {
		return fmt.Errorf("content size %d exceeds write limit %d", contentLen, p.MaxWriteBytes)
	}
	return nil
}

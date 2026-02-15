package contract

import (
	"fmt"
	"strings"
	"time"
)

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

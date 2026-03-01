package contract

import (
	"context"
	"encoding/json"
)

type MemoryProjection struct {
	Mount     VirtualMount
	Freshness string
}

type AdapterProjection struct {
	VirtualMounts []VirtualMount
	Memory        MemoryProjection
	OpaqueState   json.RawMessage
}

func (p AdapterProjection) ProjectionMounts() []VirtualMount {
	mounts := make([]VirtualMount, 0, len(p.VirtualMounts)+1)
	mounts = append(mounts, p.VirtualMounts...)
	if p.Memory.Mount != nil {
		mounts = append(mounts, p.Memory.Mount)
	}
	return mounts
}

type SessionAdapter interface {
	AdapterID() string
	CreateSession(ctx context.Context, session Session) (AdapterProjection, error)
	ResumeSession(ctx context.Context, session Session) (AdapterProjection, error)
	ObserveExecution(ctx context.Context, session Session, result ExecutionResult) (AdapterProjection, error)
	CheckpointSession(ctx context.Context, session Session) (AdapterProjection, error)
	CloseSession(ctx context.Context, session Session) (AdapterProjection, error)
}

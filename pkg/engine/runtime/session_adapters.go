package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

type adapterPhase string

const (
	adapterPhaseCreate     adapterPhase = "create"
	adapterPhaseResume     adapterPhase = "resume"
	adapterPhaseObserve    adapterPhase = "observe"
	adapterPhaseCheckpoint adapterPhase = "checkpoint"
	adapterPhaseClose      adapterPhase = "close"
)

func applySessionAdapters(ctx context.Context, session contract.Session, adapters []contract.SessionAdapter, phase adapterPhase, result contract.ExecutionResult) (contract.Session, []contract.VirtualMount, error) {
	next := session.Clone()
	mounts := make([]contract.VirtualMount, 0)
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		adapterID := strings.TrimSpace(adapter.AdapterID())
		if adapterID == "" {
			return session.Clone(), nil, fmt.Errorf("session adapter id is required")
		}
		projection, err := invokeAdapterPhase(ctx, adapter, phase, next, result)
		if err != nil {
			return session.Clone(), nil, fmt.Errorf("adapter %s %s failed: %w", adapterID, phase, err)
		}
		if len(projection.OpaqueState) > 0 {
			if next.State.Opaque == nil {
				next.State.Opaque = map[string]json.RawMessage{}
			}
			next.State.Opaque[adapterID] = append(json.RawMessage(nil), projection.OpaqueState...)
		}
		mounts = append(mounts, projection.ProjectionMounts()...)
	}
	return next, mounts, nil
}

func invokeAdapterPhase(ctx context.Context, adapter contract.SessionAdapter, phase adapterPhase, session contract.Session, result contract.ExecutionResult) (contract.AdapterProjection, error) {
	switch phase {
	case adapterPhaseCreate:
		return adapter.CreateSession(ctx, session)
	case adapterPhaseResume:
		return adapter.ResumeSession(ctx, session)
	case adapterPhaseObserve:
		return adapter.ObserveExecution(ctx, session, result)
	case adapterPhaseCheckpoint:
		return adapter.CheckpointSession(ctx, session)
	case adapterPhaseClose:
		return adapter.CloseSession(ctx, session)
	default:
		return contract.AdapterProjection{}, fmt.Errorf("unsupported adapter phase %q", phase)
	}
}

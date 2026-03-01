package engine

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/khicago/simsh/pkg/contract"
)

var executionCounter uint64

func nextExecutionID() string {
	value := atomic.AddUint64(&executionCounter, 1)
	return fmt.Sprintf("exec_%d", value)
}

func (e *Engine) ExecutePreparedResult(ctx context.Context, cmdline string, prepared PreparedOps) contract.ExecutionResult {
	startedAt := time.Now().UTC()
	collector := newExecutionTraceCollector(cmdline, prepared.Ops())
	result := contract.ExecutionResult{
		ExecutionID: nextExecutionID(),
		StartedAt:   startedAt,
		Trace:       collector.Snapshot(),
	}
	if e == nil {
		result.ExitCode = contract.ExitCodeGeneral
		result.Stdout = "execute: runtime is not initialized"
		result.FinishedAt = startedAt
		return result
	}

	normalized := prepared.Ops()
	if normalized.RequireAbsolutePath == nil {
		result.ExitCode = contract.ExitCodeGeneral
		result.Stdout = "execute: prepared ops are required"
		result.FinishedAt = time.Now().UTC()
		result.DurationMS = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
		return result
	}

	execCtx := ctx
	cancel := func() {}
	if normalized.Policy.Timeout > 0 {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > normalized.Policy.Timeout {
			var cancelFunc context.CancelFunc
			execCtx, cancelFunc = context.WithTimeout(ctx, normalized.Policy.Timeout)
			cancel = cancelFunc
		}
	}
	defer cancel()

	execCtx = collector.WithContext(execCtx)
	tracedOps := collector.WrapOps(normalized)

	stdout, code := e.runScript(execCtx, cmdline, tracedOps)
	result.ExitCode = code
	result.Stdout = stdout
	result.FinishedAt = time.Now().UTC()
	result.DurationMS = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
	result.Trace = collector.Snapshot()
	if err := execCtx.Err(); err != nil {
		result.Trace.TimedOut = result.Trace.TimedOut || errors.Is(err, context.DeadlineExceeded)
		result.Trace.Canceled = result.Trace.Canceled || errors.Is(err, context.Canceled)
	}
	return result
}

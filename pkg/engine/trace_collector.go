package engine

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/khicago/simsh/pkg/contract"
)

const traceCollectorKey contextKey = "simsh.trace.collector"

type executionTraceCollector struct {
	mu    sync.Mutex
	trace contract.ExecutionTrace
}

func newExecutionTraceCollector(commandLine string, ops contract.Ops) *executionTraceCollector {
	trace := contract.ExecutionTrace{
		CommandLine:      strings.TrimSpace(commandLine),
		EffectiveProfile: ops.Profile,
		EffectivePolicy:  ops.Policy.Clone(),
	}
	if parsed, err := parseScript(commandLine); err == nil {
		trace = populateTraceShape(trace, parsed)
	}
	return &executionTraceCollector{trace: trace}
}

func (c *executionTraceCollector) WithContext(ctx context.Context) context.Context {
	if c == nil {
		return ctx
	}
	return context.WithValue(ctx, traceCollectorKey, c)
}

func (c *executionTraceCollector) WrapOps(ops contract.Ops) contract.Ops {
	if c == nil {
		return ops
	}
	wrapped := ops

	if ops.CheckPathOp != nil {
		orig := ops.CheckPathOp
		wrapped.CheckPathOp = func(ctx context.Context, op contract.PathOp, pathValue string) error {
			c.recordRequested(pathValue)
			err := orig(ctx, op, pathValue)
			if isDeniedPathError(err) {
				c.recordDenied(pathValue)
			}
			return err
		}
	}
	if ops.ListChildren != nil {
		orig := ops.ListChildren
		wrapped.ListChildren = func(ctx context.Context, dir string) ([]string, error) {
			c.recordRequested(dir)
			children, err := orig(ctx, dir)
			if err == nil {
				c.recordRead(dir, 0)
			} else if isDeniedPathError(err) {
				c.recordDenied(dir)
			}
			return children, err
		}
	}
	if ops.IsDirPath != nil {
		orig := ops.IsDirPath
		wrapped.IsDirPath = func(ctx context.Context, pathValue string) (bool, error) {
			c.recordRequested(pathValue)
			isDir, err := orig(ctx, pathValue)
			if err == nil {
				c.recordRead(pathValue, 0)
			} else if isDeniedPathError(err) {
				c.recordDenied(pathValue)
			}
			return isDir, err
		}
	}
	if ops.DescribePath != nil {
		orig := ops.DescribePath
		wrapped.DescribePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
			c.recordRequested(pathValue)
			meta, err := orig(ctx, pathValue)
			if err == nil {
				c.recordRead(pathValue, 0)
			} else if isDeniedPathError(err) {
				c.recordDenied(pathValue)
			}
			return meta, err
		}
	}
	if ops.ReadRawContent != nil {
		orig := ops.ReadRawContent
		wrapped.ReadRawContent = func(ctx context.Context, pathValue string) (string, error) {
			c.recordRequested(pathValue)
			raw, err := orig(ctx, pathValue)
			if err == nil {
				c.recordRead(pathValue, len(raw))
			} else if isDeniedPathError(err) {
				c.recordDenied(pathValue)
			}
			return raw, err
		}
	}
	if ops.ResolveSearchPaths != nil {
		orig := ops.ResolveSearchPaths
		wrapped.ResolveSearchPaths = func(ctx context.Context, target string, recursive bool) ([]string, error) {
			c.recordRequested(target)
			paths, err := orig(ctx, target, recursive)
			if err == nil {
				c.recordRead(target, 0)
			} else if isDeniedPathError(err) {
				c.recordDenied(target)
			}
			return paths, err
		}
	}
	if ops.CollectFilesUnder != nil {
		orig := ops.CollectFilesUnder
		wrapped.CollectFilesUnder = func(ctx context.Context, target string) ([]string, error) {
			c.recordRequested(target)
			paths, err := orig(ctx, target)
			if err == nil {
				c.recordRead(target, 0)
			} else if isDeniedPathError(err) {
				c.recordDenied(target)
			}
			return paths, err
		}
	}
	if ops.WriteFile != nil {
		orig := ops.WriteFile
		wrapped.WriteFile = func(ctx context.Context, filePath string, content string) error {
			c.recordRequested(filePath)
			err := orig(ctx, filePath, content)
			if err == nil {
				c.recordWrite(filePath, len(content))
			} else if isDeniedPathError(err) {
				c.recordDenied(filePath)
			}
			return err
		}
	}
	if ops.AppendFile != nil {
		orig := ops.AppendFile
		wrapped.AppendFile = func(ctx context.Context, filePath string, content string) error {
			c.recordRequested(filePath)
			err := orig(ctx, filePath, content)
			if err == nil {
				c.recordAppend(filePath, len(content))
			} else if isDeniedPathError(err) {
				c.recordDenied(filePath)
			}
			return err
		}
	}
	if ops.EditFile != nil {
		orig := ops.EditFile
		wrapped.EditFile = func(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
			c.recordRequested(filePath)
			bytesWritten := len(newString)
			err := orig(ctx, filePath, oldString, newString, replaceAll)
			if err == nil {
				if ops.ReadRawContent != nil {
					if raw, readErr := ops.ReadRawContent(ctx, filePath); readErr == nil {
						bytesWritten = len(raw)
					}
				}
				c.recordEdit(filePath, bytesWritten)
			} else if isDeniedPathError(err) {
				c.recordDenied(filePath)
			}
			return err
		}
	}
	if ops.MakeDir != nil {
		orig := ops.MakeDir
		wrapped.MakeDir = func(ctx context.Context, dirPath string) error {
			c.recordRequested(dirPath)
			err := orig(ctx, dirPath)
			if err == nil {
				c.recordCreatedDir(dirPath)
			} else if isDeniedPathError(err) {
				c.recordDenied(dirPath)
			}
			return err
		}
	}
	if ops.RemoveFile != nil {
		orig := ops.RemoveFile
		wrapped.RemoveFile = func(ctx context.Context, filePath string) error {
			c.recordRequested(filePath)
			err := orig(ctx, filePath)
			if err == nil {
				c.recordRemoved(filePath)
			} else if isDeniedPathError(err) {
				c.recordDenied(filePath)
			}
			return err
		}
	}
	if ops.RemoveDir != nil {
		orig := ops.RemoveDir
		wrapped.RemoveDir = func(ctx context.Context, dirPath string) error {
			c.recordRequested(dirPath)
			err := orig(ctx, dirPath)
			if err == nil {
				c.recordRemoved(dirPath)
			} else if isDeniedPathError(err) {
				c.recordDenied(dirPath)
			}
			return err
		}
	}
	if ops.RunExternalCommand != nil {
		orig := ops.RunExternalCommand
		wrapped.RunExternalCommand = func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
			if strings.HasPrefix(strings.TrimSpace(req.RawPath), "/") {
				c.recordRequested(req.RawPath)
			}
			result, err := orig(ctx, req)
			if err == nil {
				c.recordExternalStdout(len(result.Stdout))
			}
			return result, err
		}
	}

	return wrapped
}

func (c *executionTraceCollector) Snapshot() contract.ExecutionTrace {
	if c == nil {
		return contract.ExecutionTrace{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	trace := c.trace
	trace.Argv = append([]string(nil), c.trace.Argv...)
	if len(c.trace.Pipeline) > 0 {
		trace.Pipeline = make([]contract.ExecutionTraceStep, 0, len(c.trace.Pipeline))
		for _, step := range c.trace.Pipeline {
			trace.Pipeline = append(trace.Pipeline, contract.ExecutionTraceStep{
				Command: step.Command,
				Argv:    append([]string(nil), step.Argv...),
			})
		}
	}
	trace.RequestedPaths = append([]string(nil), c.trace.RequestedPaths...)
	trace.ReadPaths = append([]string(nil), c.trace.ReadPaths...)
	trace.WrittenPaths = append([]string(nil), c.trace.WrittenPaths...)
	trace.AppendedPaths = append([]string(nil), c.trace.AppendedPaths...)
	trace.EditedPaths = append([]string(nil), c.trace.EditedPaths...)
	trace.CreatedDirs = append([]string(nil), c.trace.CreatedDirs...)
	trace.RemovedPaths = append([]string(nil), c.trace.RemovedPaths...)
	trace.DeniedPaths = append([]string(nil), c.trace.DeniedPaths...)
	trace.EffectivePolicy = c.trace.EffectivePolicy.Clone()
	return trace
}

func (c *executionTraceCollector) MarkOutputTruncated() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace.OutputTruncated = true
}

func (c *executionTraceCollector) recordRequested(pathValue string) {
	c.recordPath(&c.trace.RequestedPaths, pathValue)
}

func (c *executionTraceCollector) recordRead(pathValue string, bytes int) {
	c.recordPath(&c.trace.ReadPaths, pathValue)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace.BytesRead += bytes
}

func (c *executionTraceCollector) recordWrite(pathValue string, bytes int) {
	c.recordPath(&c.trace.WrittenPaths, pathValue)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace.BytesWritten += bytes
}

func (c *executionTraceCollector) recordAppend(pathValue string, bytes int) {
	c.recordPath(&c.trace.AppendedPaths, pathValue)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace.BytesWritten += bytes
}

func (c *executionTraceCollector) recordEdit(pathValue string, bytes int) {
	c.recordPath(&c.trace.EditedPaths, pathValue)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace.BytesWritten += bytes
}

func (c *executionTraceCollector) recordCreatedDir(pathValue string) {
	c.recordPath(&c.trace.CreatedDirs, pathValue)
}

func (c *executionTraceCollector) recordRemoved(pathValue string) {
	c.recordPath(&c.trace.RemovedPaths, pathValue)
}

func (c *executionTraceCollector) recordDenied(pathValue string) {
	c.recordPath(&c.trace.DeniedPaths, pathValue)
}

func (c *executionTraceCollector) recordExternalStdout(bytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace.BytesRead += bytes
}

func (c *executionTraceCollector) recordPath(target *[]string, pathValue string) {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" || pathValue == "/" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, existing := range *target {
		if existing == pathValue {
			return
		}
	}
	*target = append(*target, pathValue)
}

func populateTraceShape(trace contract.ExecutionTrace, statements []parsedStatement) contract.ExecutionTrace {
	for _, statement := range statements {
		for _, node := range statement.pipelines {
			for _, command := range node.pipeline.commands {
				if len(command.args) == 0 {
					continue
				}
				step := contract.ExecutionTraceStep{
					Command: strings.TrimSpace(command.args[0]),
					Argv:    append([]string(nil), command.args...),
				}
				trace.Pipeline = append(trace.Pipeline, step)
				if trace.Command == "" {
					trace.Command = step.Command
					trace.Argv = append([]string(nil), step.Argv...)
				}
			}
		}
	}
	return trace
}

func traceCollectorFromContext(ctx context.Context) *executionTraceCollector {
	collector, _ := ctx.Value(traceCollectorKey).(*executionTraceCollector)
	return collector
}

func markTraceOutputTruncated(ctx context.Context) {
	if collector := traceCollectorFromContext(ctx); collector != nil {
		collector.MarkOutputTruncated()
	}
}

func markTraceRequestedPath(ctx context.Context, pathValue string) {
	if collector := traceCollectorFromContext(ctx); collector != nil {
		collector.recordRequested(pathValue)
	}
}

func markTraceDeniedPath(ctx context.Context, pathValue string) {
	if collector := traceCollectorFromContext(ctx); collector != nil {
		collector.recordDenied(pathValue)
	}
}

func isDeniedPathError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, contract.ErrUnsupported) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "no such file") || strings.Contains(message, "not found") {
		return false
	}
	return strings.Contains(message, "not allowed") ||
		strings.Contains(message, "not supported") ||
		strings.Contains(message, "permission") ||
		strings.Contains(message, "denied") ||
		strings.Contains(message, "read-only") ||
		strings.Contains(message, "access")
}

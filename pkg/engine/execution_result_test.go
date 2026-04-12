package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestEngineExecuteResultStructuredFields(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.DefaultPolicy()

	result := eng.ExecuteResult(context.Background(), "echo hello", ops)
	if result.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", result)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello" || result.Stderr != "" {
		t.Fatalf("unexpected stdout/stderr: %+v", result)
	}
	if result.Trace.CommandLine != "echo hello" {
		t.Fatalf("unexpected trace: %+v", result.Trace)
	}
	if result.Trace.Command != "echo" || len(result.Trace.Pipeline) != 1 {
		t.Fatalf("unexpected command shape: %+v", result.Trace)
	}
	if len(result.Trace.Executed) != 1 || result.Trace.Executed[0].Command != "echo" || !result.Trace.Executed[0].Executed {
		t.Fatalf("unexpected executed trace shape: %+v", result.Trace.Executed)
	}
	if result.Trace.EffectiveProfile != contract.ProfileCoreStrict {
		t.Fatalf("unexpected effective profile: %+v", result.Trace)
	}
	if result.Trace.EffectivePolicy.WriteMode != contract.WriteModeReadOnly {
		t.Fatalf("unexpected effective policy: %+v", result.Trace.EffectivePolicy)
	}
}

func TestEngineExecuteResultTracePaths(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	result := eng.ExecuteResult(context.Background(), "echo hello > /workspace/out.txt; cat /workspace/out.txt", ops)
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit_code=%d stdout=%q", result.ExitCode, result.Stdout)
	}
	if !containsTracePath(result.Trace.RequestedPaths, "/workspace/out.txt") {
		t.Fatalf("expected requested path in trace: %+v", result.Trace)
	}
	if !containsTracePath(result.Trace.WrittenPaths, "/workspace/out.txt") {
		t.Fatalf("expected written path in trace: %+v", result.Trace)
	}
	if !containsTracePath(result.Trace.ReadPaths, "/workspace/out.txt") {
		t.Fatalf("expected read path in trace: %+v", result.Trace)
	}
	if result.Trace.BytesWritten == 0 || result.Trace.BytesRead == 0 {
		t.Fatalf("expected resource summary bytes in trace: %+v", result.Trace)
	}
}

func TestEngineExecuteResultTraceNormalizesRelativeRedirectionPaths(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	workspaceDir := "/" + "workspace"
	notePath := workspaceDir + "/" + "note.txt"
	result := eng.ExecuteResult(context.Background(), "cd "+workspaceDir+"; echo hello > note.txt", ops)
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit_code=%d stdout=%q", result.ExitCode, result.Stdout)
	}
	if containsTracePath(result.Trace.RequestedPaths, "note.txt") {
		t.Fatalf("requested_paths unexpectedly contains raw relative path: %+v", result.Trace)
	}
	if !containsTracePath(result.Trace.RequestedPaths, notePath) {
		t.Fatalf("expected resolved requested path in trace: %+v", result.Trace)
	}
	if !containsTracePath(result.Trace.WrittenPaths, notePath) {
		t.Fatalf("expected resolved written path in trace: %+v", result.Trace)
	}
}

func TestEngineExecuteResultTraceAppendAndRemove(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	appended := eng.ExecuteResult(context.Background(), "echo hello | tee -a /workspace/todo.txt", ops)
	if appended.ExitCode != 0 {
		t.Fatalf("unexpected append exit_code=%d stdout=%q", appended.ExitCode, appended.Stdout)
	}
	if !containsTracePath(appended.Trace.AppendedPaths, "/workspace/todo.txt") {
		t.Fatalf("expected appended path in trace: %+v", appended.Trace)
	}
	if appended.Trace.BytesWritten != len("hello") {
		t.Fatalf("append bytes_written=%d, want %d trace=%+v", appended.Trace.BytesWritten, len("hello"), appended.Trace)
	}

	removed := eng.ExecuteResult(context.Background(), "rm /workspace/todo.txt", ops)
	if removed.ExitCode != 0 {
		t.Fatalf("unexpected remove exit_code=%d stdout=%q", removed.ExitCode, removed.Stdout)
	}
	if !containsTracePath(removed.Trace.RemovedPaths, "/workspace/todo.txt") {
		t.Fatalf("expected removed path in trace: %+v", removed.Trace)
	}
}

func TestEngineExecuteResultTraceEditWritesFinalFileBytes(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	result := eng.ExecuteResult(context.Background(), "sed -i 's/hello/hi/' /workspace/readme.md", ops)
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit_code=%d stdout=%q", result.ExitCode, result.Stdout)
	}
	if !containsTracePath(result.Trace.EditedPaths, "/workspace/readme.md") {
		t.Fatalf("expected edited path in trace: %+v", result.Trace)
	}
	if result.Trace.BytesWritten != len("hi\nworld\n") {
		t.Fatalf("bytes_written=%d, want %d trace=%+v", result.Trace.BytesWritten, len("hi\nworld\n"), result.Trace)
	}
}

func TestEngineExecuteResultTraceDeniedAndOutputLimit(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly, MaxOutputBytes: 1024}

	denied := eng.ExecuteResult(context.Background(), "echo hello > /workspace/out.txt", ops)
	if !containsTracePath(denied.Trace.DeniedPaths, "/workspace/out.txt") {
		t.Fatalf("expected denied path in trace: %+v", denied.Trace)
	}

	ops.Policy = contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly, MaxOutputBytes: 4}
	truncated := eng.ExecuteResult(context.Background(), "echo hello", ops)
	if !truncated.Trace.OutputTruncated {
		t.Fatalf("expected output_truncated trace flag: %+v", truncated.Trace)
	}
	if truncated.Stdout != "hell" || truncated.Stderr != "" {
		t.Fatalf("unexpected truncated stdout/stderr: %+v", truncated)
	}
	if truncated.ExitCode != contract.ExitCodeGeneral {
		t.Fatalf("unexpected truncated exit_code=%d, want %d", truncated.ExitCode, contract.ExitCodeGeneral)
	}
	if !utf8.ValidString(truncated.Stdout) || !utf8.ValidString(truncated.Stderr) {
		t.Fatalf("truncated output must stay valid UTF-8: %+v", truncated)
	}

	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly, MaxOutputBytes: 4}
	pipelineTruncated := eng.ExecuteResult(context.Background(), "echo hello | cat", ops)
	if !pipelineTruncated.Trace.OutputTruncated {
		t.Fatalf("expected pipeline output_truncated trace flag: %+v", pipelineTruncated.Trace)
	}
	if pipelineTruncated.Stdout == "" {
		t.Fatalf("expected truncated pipeline stdout to preserve partial content: %+v", pipelineTruncated)
	}
	if pipelineTruncated.ExitCode != contract.ExitCodeGeneral {
		t.Fatalf("unexpected pipeline truncated exit_code=%d, want %d", pipelineTruncated.ExitCode, contract.ExitCodeGeneral)
	}
	if pipelineTruncated.Stdout == "" && pipelineTruncated.Stderr == "" {
		t.Fatalf("expected pipeline truncation to preserve partial channel truth: %+v", pipelineTruncated)
	}
}

func TestEngineExecuteResultTraceNormalizesRelativeDeniedRedirectionPaths(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)

	workspaceDir := "/" + "workspace"
	outPath := workspaceDir + "/" + "out.txt"
	appendPath := workspaceDir + "/" + "append.txt"
	largePath := workspaceDir + "/" + "large.txt"

	t.Run("policy denial", func(t *testing.T) {
		ops := contract.OpsFromFilesystem(newTestFS())
		ops.Profile = contract.ProfileBashPlus
		ops.Policy = contract.DefaultPolicy()

		result := eng.ExecuteResult(context.Background(), "cd "+workspaceDir+"; echo hello > out.txt", ops)
		if result.ExitCode == 0 {
			t.Fatalf("expected denied redirection, got %+v", result)
		}
		if containsTracePath(result.Trace.DeniedPaths, "out.txt") {
			t.Fatalf("denied_paths unexpectedly contains raw relative path: %+v", result.Trace)
		}
		if !containsTracePath(result.Trace.DeniedPaths, outPath) {
			t.Fatalf("expected resolved denied path in trace: %+v", result.Trace)
		}
	})

	t.Run("append unsupported", func(t *testing.T) {
		fs := newTestFS()
		ops := contract.OpsFromFilesystem(fs)
		ops.Profile = contract.ProfileBashPlus
		ops.Policy = contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		}
		ops.AppendFile = nil

		result := eng.ExecuteResult(context.Background(), "cd "+workspaceDir+"; echo hello >> append.txt", ops)
		if result.ExitCode == 0 {
			t.Fatalf("expected append redirection failure, got %+v", result)
		}
		if containsTracePath(result.Trace.DeniedPaths, "append.txt") {
			t.Fatalf("denied_paths unexpectedly contains raw relative path: %+v", result.Trace)
		}
		if !containsTracePath(result.Trace.DeniedPaths, appendPath) {
			t.Fatalf("expected resolved denied path in trace: %+v", result.Trace)
		}
	})

	t.Run("write limited payload denial", func(t *testing.T) {
		ops := contract.OpsFromFilesystem(newTestFS())
		ops.Profile = contract.ProfileBashPlus
		ops.Policy = contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeWriteLimited,
			MaxWriteBytes:    1,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		}

		result := eng.ExecuteResult(context.Background(), "cd "+workspaceDir+"; echo hello > large.txt", ops)
		if result.ExitCode == 0 {
			t.Fatalf("expected write-limited redirection failure, got %+v", result)
		}
		if containsTracePath(result.Trace.DeniedPaths, "large.txt") {
			t.Fatalf("denied_paths unexpectedly contains raw relative path: %+v", result.Trace)
		}
		if !containsTracePath(result.Trace.DeniedPaths, largePath) {
			t.Fatalf("expected resolved denied path in trace: %+v", result.Trace)
		}
	})
}

func TestEngineExecuteResultTraceMutationDenials(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.DefaultPolicy()

	deniedEditByPolicy := eng.ExecuteResult(context.Background(), "sed -i 's/hello/bye/' /workspace/readme.md", ops)
	if !containsTracePath(deniedEditByPolicy.Trace.DeniedPaths, "/workspace/readme.md") {
		t.Fatalf("expected edit policy denial path in trace: %+v", deniedEditByPolicy.Trace)
	}
	deniedRemoveByPolicy := eng.ExecuteResult(context.Background(), "rm /workspace/todo.txt", ops)
	if !containsTracePath(deniedRemoveByPolicy.Trace.DeniedPaths, "/workspace/todo.txt") {
		t.Fatalf("expected remove policy denial path in trace: %+v", deniedRemoveByPolicy.Trace)
	}
	deniedWriteByPolicy := eng.ExecuteResult(context.Background(), "echo hello | tee /workspace/out.txt", ops)
	if !containsTracePath(deniedWriteByPolicy.Trace.DeniedPaths, "/workspace/out.txt") {
		t.Fatalf("expected write policy denial path in trace: %+v", deniedWriteByPolicy.Trace)
	}

	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	deniedEdit := eng.ExecuteResult(context.Background(), "sed -i 's/ls/echo/' /sys/bin/ls", ops)
	if !containsTracePath(deniedEdit.Trace.DeniedPaths, "/sys/bin/ls") {
		t.Fatalf("expected edit denial path in trace: %+v", deniedEdit.Trace)
	}
	deniedMount := eng.ExecuteResult(context.Background(), "touch /sys/bin/new.txt", ops)
	if !containsTracePath(deniedMount.Trace.DeniedPaths, "/sys/bin/new.txt") {
		t.Fatalf("expected mounted path denial in trace: %+v", deniedMount.Trace)
	}
}

func TestEngineExecuteResultPreservesExternalCommandStderr(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.DefaultPolicy()

	warn := eng.ExecuteResult(context.Background(), "report_tool --warn", ops)
	if warn.ExitCode != 0 {
		t.Fatalf("unexpected exit_code=%d result=%+v", warn.ExitCode, warn)
	}
	if warn.Stdout != "report ok" || warn.Stderr != "report warning" {
		t.Fatalf("unexpected external stdout/stderr: %+v", warn)
	}
	if warn.Trace.ExternalStdoutBytes != len("report ok") || warn.Trace.ExternalStderrBytes != len("report warning") {
		t.Fatalf("unexpected external trace bytes: %+v", warn.Trace)
	}
	if len(warn.Trace.Executed) == 0 || warn.Trace.Executed[0].Namespace != contract.CommandNamespaceExternal {
		t.Fatalf("unexpected executed external trace: %+v", warn.Trace.Executed)
	}
	if len(warn.Trace.ExternalOutcomes) == 0 || warn.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeSuccess {
		t.Fatalf("unexpected successful external outcome kind: %+v", warn.Trace.ExternalOutcomes)
	}

	fail := eng.ExecuteResult(context.Background(), "report_tool --fail", ops)
	if fail.ExitCode != 17 {
		t.Fatalf("unexpected exit_code=%d result=%+v", fail.ExitCode, fail)
	}
	if fail.Stdout != "" || fail.Stderr != "report failed" {
		t.Fatalf("unexpected failing external stdout/stderr: %+v", fail)
	}
	if fail.FlattenOutput() != "report failed" {
		t.Fatalf("unexpected flattened external failure output: %q", fail.FlattenOutput())
	}
	if fail.Trace.ExternalStdoutBytes != 0 || fail.Trace.ExternalStderrBytes != len("report failed") {
		t.Fatalf("unexpected failing external trace bytes: %+v", fail.Trace)
	}
	if len(fail.Trace.Executed) == 0 || fail.Trace.Executed[0].Namespace != contract.CommandNamespaceExternal {
		t.Fatalf("unexpected failing executed external trace: %+v", fail.Trace.Executed)
	}
	if len(fail.Trace.ExternalOutcomes) == 0 || fail.Trace.ExternalOutcomes[0].ExitCode == nil || *fail.Trace.ExternalOutcomes[0].ExitCode != 17 || fail.Trace.ExternalOutcomes[0].RawExitCode != nil {
		t.Fatalf("unexpected failing external outcome trace: %+v", fail.Trace.ExternalOutcomes)
	}
	if fail.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeNonZeroExit {
		t.Fatalf("unexpected failing external outcome kind: %+v", fail.Trace.ExternalOutcomes[0])
	}
}

func TestEngineExecuteResultTraceExecutedSkipsConditionalBranches(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.DefaultPolicy()
	missingPath := strings.Join([]string{"", "missing"}, "/")

	result := eng.ExecuteResult(context.Background(), "cat "+missingPath+" && echo no; echo yes", ops)
	if len(result.Trace.Pipeline) < 3 {
		t.Fatalf("unexpected parsed pipeline shape: %+v", result.Trace.Pipeline)
	}
	if len(result.Trace.Executed) != 2 {
		t.Fatalf("executed trace = %+v, want 2 executed steps", result.Trace.Executed)
	}
	if result.Trace.Executed[0].Command != "cat" || result.Trace.Executed[1].Command != "echo" {
		t.Fatalf("executed trace order = %+v", result.Trace.Executed)
	}
}

func TestEngineExecuteResultTraceExecutedExcludesExternalProviderFailures(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.DefaultPolicy()

	result := eng.ExecuteResult(context.Background(), "unknown_external_cmd", ops)
	if len(result.Trace.Executed) != 0 {
		t.Fatalf("executed trace = %+v, want no executed external step on provider failure", result.Trace.Executed)
	}
	if result.Stderr == "" && result.Stdout == "" {
		t.Fatalf("result lost provider failure output: %+v", result)
	}
	if len(result.Trace.ExternalOutcomes) != 1 {
		t.Fatalf("external_outcomes = %+v, want 1 provider failure outcome", result.Trace.ExternalOutcomes)
	}
	if result.Trace.ExternalOutcomes[0].ProviderError == "" {
		t.Fatalf("provider failure outcome lost provider_error: %+v", result.Trace.ExternalOutcomes[0])
	}
	if result.Trace.ExternalOutcomes[0].ExitCode == nil || *result.Trace.ExternalOutcomes[0].ExitCode != contract.ExitCodeUnsupported {
		t.Fatalf("provider failure outcome lost compatibility exit code alignment: %+v", result.Trace.ExternalOutcomes[0])
	}
	if result.Trace.ExternalOutcomes[0].ResolvedPath == "" {
		t.Fatalf("provider failure outcome lost resolved path: %+v", result.Trace.ExternalOutcomes[0])
	}
	if result.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeUnsupported {
		t.Fatalf("provider failure outcome kind = %q, want %q", result.Trace.ExternalOutcomes[0].OutcomeKind, contract.ExternalOutcomeUnsupported)
	}
}

func TestEngineExecuteResultRecordsUnsupportedExternalOutcome(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.DefaultPolicy()
	ops.RunExternalCommand = nil

	result := eng.ExecuteResult(context.Background(), "report_tool", ops)
	if result.ExitCode != contract.ExitCodeUnsupported {
		t.Fatalf("unsupported external exit_code=%d, want %d", result.ExitCode, contract.ExitCodeUnsupported)
	}
	if len(result.Trace.ExternalOutcomes) != 1 {
		t.Fatalf("external_outcomes = %+v, want 1 unsupported external outcome", result.Trace.ExternalOutcomes)
	}
	if result.Trace.ExternalOutcomes[0].ExitCode == nil || *result.Trace.ExternalOutcomes[0].ExitCode != contract.ExitCodeUnsupported {
		t.Fatalf("unsupported external outcome exit_code mismatch: %+v", result.Trace.ExternalOutcomes[0])
	}
	if result.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeUnsupported {
		t.Fatalf("unsupported external outcome kind = %q, want %q", result.Trace.ExternalOutcomes[0].OutcomeKind, contract.ExternalOutcomeUnsupported)
	}
}

func TestEngineExecuteResultDoesNotInferExternalCommandNotFoundFromUnsupported(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.DefaultPolicy()
	ops.RootDir = ""
	ops.RequireAbsolutePath = func(raw string) (string, error) {
		return raw, nil
	}

	externalPath := contract.VirtualExternalBinDir + "/" + "missing_tool"
	result := eng.ExecuteResult(context.Background(), externalPath, ops)
	if result.ExitCode != contract.ExitCodeUnsupported {
		t.Fatalf("pathlike external unsupported exit_code=%d, want %d", result.ExitCode, contract.ExitCodeUnsupported)
	}
	if len(result.Trace.ExternalOutcomes) != 1 {
		t.Fatalf("external_outcomes = %+v, want 1 unsupported outcome", result.Trace.ExternalOutcomes)
	}
	if result.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeUnsupported {
		t.Fatalf("pathlike external outcome kind = %q, want %q", result.Trace.ExternalOutcomes[0].OutcomeKind, contract.ExternalOutcomeUnsupported)
	}
}

func TestEngineExecuteResultClassifiesExternalCommandNotFound(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.DefaultPolicy()
	ops.RunExternalCommand = func(context.Context, contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
		return contract.ExternalCommandResult{ProviderError: "missing command"}, contract.ErrExternalCommandNotFound
	}

	result := eng.ExecuteResult(context.Background(), "missing_tool", ops)
	if result.ExitCode != contract.ExitCodeGeneral {
		t.Fatalf("external command not found exit_code=%d, want %d", result.ExitCode, contract.ExitCodeGeneral)
	}
	if len(result.Trace.ExternalOutcomes) != 1 {
		t.Fatalf("external_outcomes = %+v, want 1 command-not-found outcome", result.Trace.ExternalOutcomes)
	}
	if result.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeCommandNotFound {
		t.Fatalf("external command not found outcome kind = %q, want %q", result.Trace.ExternalOutcomes[0].OutcomeKind, contract.ExternalOutcomeCommandNotFound)
	}
}

func TestEngineExecuteResultClassifiesExternalProviderFailure(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.DefaultPolicy()
	ops.RunExternalCommand = func(context.Context, contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
		return contract.ExternalCommandResult{ProviderError: "provider unavailable"}, errors.New("provider unavailable")
	}

	result := eng.ExecuteResult(context.Background(), "report_tool", ops)
	if result.ExitCode != contract.ExitCodeGeneral {
		t.Fatalf("provider failure exit_code=%d, want %d", result.ExitCode, contract.ExitCodeGeneral)
	}
	if len(result.Trace.ExternalOutcomes) != 1 {
		t.Fatalf("external_outcomes = %+v, want 1 provider failure outcome", result.Trace.ExternalOutcomes)
	}
	if result.Trace.ExternalOutcomes[0].OutcomeKind != contract.ExternalOutcomeProviderFailure {
		t.Fatalf("provider failure outcome kind = %q, want %q", result.Trace.ExternalOutcomes[0].OutcomeKind, contract.ExternalOutcomeProviderFailure)
	}
}

func TestEngineExecuteResultClassifiesExternalContextErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want contract.ExternalOutcomeKind
	}{
		{name: "canceled", err: context.Canceled, want: contract.ExternalOutcomeCanceled},
		{name: "timed_out", err: context.DeadlineExceeded, want: contract.ExternalOutcomeTimedOut},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := engine.NewRegistry()
			builtin.RegisterDefaults(registry)
			eng := engine.New(registry)
			ops := contract.OpsFromFilesystem(newTestFS())
			ops.Profile = contract.ProfileBashPlus
			ops.Policy = contract.DefaultPolicy()
			ops.RunExternalCommand = func(context.Context, contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
				return contract.ExternalCommandResult{}, tc.err
			}

			result := eng.ExecuteResult(context.Background(), "report_tool", ops)
			if result.ExitCode != contract.ExitCodeGeneral {
				t.Fatalf("ExecuteResult(report_tool) exit_code=%d, want %d", result.ExitCode, contract.ExitCodeGeneral)
			}
			if len(result.Trace.ExternalOutcomes) != 1 {
				t.Fatalf("ExecuteResult(report_tool).Trace.ExternalOutcomes = %+v, want 1 outcome", result.Trace.ExternalOutcomes)
			}
			if result.Trace.ExternalOutcomes[0].OutcomeKind != tc.want {
				t.Fatalf("ExecuteResult(report_tool) outcome kind = %q, want %q", result.Trace.ExternalOutcomes[0].OutcomeKind, tc.want)
			}
		})
	}
}

func TestEngineExecuteResultMarksCanceledContext(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.DefaultPolicy()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := eng.ExecuteResult(ctx, "cat /workspace/readme.md", ops)
	if result.ExitCode == 0 {
		t.Fatalf("expected canceled execution to fail: %+v", result)
	}
	if !result.Trace.Canceled {
		t.Fatalf("expected canceled trace flag: %+v", result.Trace)
	}
}

func TestEngineExecuteResultMarksTimedOutContext(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.DefaultPolicy()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	result := eng.ExecuteResult(ctx, "cat /workspace/readme.md", ops)
	if result.ExitCode == 0 {
		t.Fatalf("expected timed out execution to fail: %+v", result)
	}
	if !result.Trace.TimedOut {
		t.Fatalf("expected timed_out trace flag: %+v", result.Trace)
	}
}

func containsTracePath(paths []string, target string) bool {
	for _, pathValue := range paths {
		if pathValue == target {
			return true
		}
	}
	return false
}

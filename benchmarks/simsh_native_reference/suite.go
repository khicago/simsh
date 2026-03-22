package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/khicago/simsh/pkg/contract"
	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
	"github.com/khicago/simsh/pkg/fs"
)

type GateThresholds struct {
	TraceCompleteness        float64 `json:"trace_completeness"`
	SessionSuccess           float64 `json:"session_success"`
	ReviewablePatchLatencyMS int64   `json:"reviewable_patch_latency_ms"`
	AsyncCompletionSuccess   float64 `json:"async_completion_success"`
}

type GateMetrics struct {
	TraceCompleteness        float64 `json:"trace_completeness"`
	SessionSuccess           float64 `json:"session_success"`
	ReviewablePatchLatencyMS int64   `json:"reviewable_patch_latency_ms"`
	AsyncCompletionSuccess   float64 `json:"async_completion_success"`
}

type GateResult struct {
	Name      string  `json:"name"`
	Actual    float64 `json:"actual"`
	Threshold float64 `json:"threshold"`
	Pass      bool    `json:"pass"`
}

type ScenarioReport struct {
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Success           bool     `json:"success"`
	SessionScoped     bool     `json:"session_scoped"`
	AsyncCandidate    bool     `json:"async_candidate"`
	PatchWorkflow     bool     `json:"patch_workflow"`
	DurationMS        int64    `json:"duration_ms"`
	TraceChecksPassed int      `json:"trace_checks_passed"`
	TraceChecksTotal  int      `json:"trace_checks_total"`
	TraceCompleteness float64  `json:"trace_completeness"`
	Notes             []string `json:"notes,omitempty"`
}

type SuiteReport struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Thresholds  GateThresholds   `json:"thresholds"`
	Metrics     GateMetrics      `json:"metrics"`
	Gates       []GateResult     `json:"gates"`
	Scenarios   []ScenarioReport `json:"scenarios"`
}

type traceExpectation struct {
	requested    []string
	read         []string
	written      []string
	edited       []string
	denied       []string
	bytesRead    func(int) bool
	bytesWritten func(int) bool
	canceled     *bool
	timedOut     *bool
}

func defaultThresholds() GateThresholds {
	return GateThresholds{
		TraceCompleteness:        0.90,
		SessionSuccess:           0.80,
		ReviewablePatchLatencyMS: 15 * 60 * 1000,
		AsyncCompletionSuccess:   0.60,
	}
}

func runSuite() (SuiteReport, error) {
	scenarios := []func() (ScenarioReport, error){
		runRelativeNavigationScenario,
		runInspectEditWriteScenario,
		runMountBoundaryScenario,
		runCommandNamespaceScenario,
		runTracePlanningScenario,
		runCancelTimeoutScenario,
	}

	report := SuiteReport{
		GeneratedAt: time.Now().UTC(),
		Thresholds:  defaultThresholds(),
		Scenarios:   make([]ScenarioReport, 0, len(scenarios)),
	}
	for _, run := range scenarios {
		scenario, err := run()
		if err != nil {
			return SuiteReport{}, err
		}
		report.Scenarios = append(report.Scenarios, scenario)
	}

	report.Metrics = computeMetrics(report.Scenarios)
	report.Gates = evaluateGates(report.Metrics, report.Thresholds)
	return report, nil
}

func computeMetrics(scenarios []ScenarioReport) GateMetrics {
	var tracePassed, traceTotal int
	var sessionPassed, sessionTotal int
	var asyncPassed, asyncTotal int
	patchDurations := make([]int64, 0)

	for _, scenario := range scenarios {
		tracePassed += scenario.TraceChecksPassed
		traceTotal += scenario.TraceChecksTotal
		if scenario.SessionScoped {
			sessionTotal++
			if scenario.Success {
				sessionPassed++
			}
		}
		if scenario.AsyncCandidate {
			asyncTotal++
			if scenario.Success {
				asyncPassed++
			}
		}
		if scenario.PatchWorkflow {
			patchDurations = append(patchDurations, scenario.DurationMS)
		}
	}

	metrics := GateMetrics{}
	if traceTotal > 0 {
		metrics.TraceCompleteness = float64(tracePassed) / float64(traceTotal)
	} else {
		metrics.TraceCompleteness = 1.0
	}
	if sessionTotal > 0 {
		metrics.SessionSuccess = float64(sessionPassed) / float64(sessionTotal)
	} else {
		metrics.SessionSuccess = 1.0
	}
	if asyncTotal > 0 {
		metrics.AsyncCompletionSuccess = float64(asyncPassed) / float64(asyncTotal)
	} else {
		metrics.AsyncCompletionSuccess = 1.0
	}
	if len(patchDurations) > 0 {
		slices.Sort(patchDurations)
		metrics.ReviewablePatchLatencyMS = patchDurations[len(patchDurations)/2]
	}
	return metrics
}

func evaluateGates(metrics GateMetrics, thresholds GateThresholds) []GateResult {
	return []GateResult{
		{
			Name:      "trace_completeness",
			Actual:    metrics.TraceCompleteness,
			Threshold: thresholds.TraceCompleteness,
			Pass:      metrics.TraceCompleteness >= thresholds.TraceCompleteness,
		},
		{
			Name:      "session_success",
			Actual:    metrics.SessionSuccess,
			Threshold: thresholds.SessionSuccess,
			Pass:      metrics.SessionSuccess >= thresholds.SessionSuccess,
		},
		{
			Name:      "reviewable_patch_latency_ms",
			Actual:    float64(metrics.ReviewablePatchLatencyMS),
			Threshold: float64(thresholds.ReviewablePatchLatencyMS),
			Pass:      metrics.ReviewablePatchLatencyMS <= thresholds.ReviewablePatchLatencyMS,
		},
		{
			Name:      "async_completion_success",
			Actual:    metrics.AsyncCompletionSuccess,
			Threshold: thresholds.AsyncCompletionSuccess,
			Pass:      metrics.AsyncCompletionSuccess >= thresholds.AsyncCompletionSuccess,
		},
	}
}

func runRelativeNavigationScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)

	manager := newFullSessionManager()
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullPolicy(),
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	setup, err := manager.Execute(context.Background(), session.SessionID, "mkdir -p /task_outputs/project/docs && echo hello > /task_outputs/project/docs/readme.md && cd /task_outputs/project", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	pwdResult, err := manager.Execute(context.Background(), session.SessionID, "pwd", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	readResult, err := manager.Execute(context.Background(), session.SessionID, "cat docs/readme.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}

	tracePassed, traceTotal := evaluateTrace(readResult.Result.Trace, traceExpectation{
		requested: []string{"/task_outputs/project/docs/readme.md"},
		read:      []string{"/task_outputs/project/docs/readme.md"},
	})
	notes := []string{}
	success := setup.Result.ExitCode == 0 &&
		pwdResult.Result.ExitCode == 0 &&
		strings.TrimSpace(pwdResult.Result.Stdout) == "/task_outputs/project" &&
		readResult.Result.ExitCode == 0 &&
		strings.TrimSpace(readResult.Result.Stdout) == "hello" &&
		readResult.Session.State.WorkingDir == "/task_outputs/project"
	if !success {
		notes = append(notes, "session-relative navigation or cwd persistence failed")
	}
	duration := time.Since(start).Milliseconds()
	return ScenarioReport{
		Name:              "relative_navigation_session",
		Category:          "relative_path_navigation",
		Success:           success,
		SessionScoped:     true,
		AsyncCandidate:    false,
		PatchWorkflow:     false,
		DurationMS:        duration,
		TraceChecksPassed: tracePassed,
		TraceChecksTotal:  traceTotal,
		TraceCompleteness: ratio(tracePassed, traceTotal),
		Notes:             notes,
	}, nil
}

func runInspectEditWriteScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)
	reportPath := filepath.Join(root, "task_outputs", "report.md")
	if err := os.WriteFile(reportPath, []byte("alpha\n"), 0o644); err != nil {
		return ScenarioReport{}, err
	}

	stack, err := runtimeengine.New(runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullPolicy(),
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	result := stack.ExecuteResult(context.Background(), "sed -i 's/alpha/beta/' /task_outputs/report.md; cat /task_outputs/report.md")
	tracePassed, traceTotal := evaluateTrace(result.Trace, traceExpectation{
		requested: []string{"/task_outputs/report.md"},
		edited:    []string{"/task_outputs/report.md"},
		read:      []string{"/task_outputs/report.md"},
		bytesWritten: func(v int) bool {
			return v >= len("beta\n")
		},
	})
	notes := []string{}
	success := result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "beta"
	if !success {
		notes = append(notes, "inspect/edit/write loop did not produce final reviewable output")
	}
	return ScenarioReport{
		Name:              "inspect_edit_write_loop",
		Category:          "file_inspect_edit_write_loops",
		Success:           success,
		SessionScoped:     true,
		AsyncCandidate:    true,
		PatchWorkflow:     true,
		DurationMS:        time.Since(start).Milliseconds(),
		TraceChecksPassed: tracePassed,
		TraceChecksTotal:  traceTotal,
		TraceCompleteness: ratio(tracePassed, traceTotal),
		Notes:             notes,
	}, nil
}

func runMountBoundaryScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)

	stack, err := runtimeengine.New(runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullPolicy(),
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	result := stack.ExecuteResult(context.Background(), "cd /task_outputs; touch ../sys/bin/new.txt")
	tracePassed, traceTotal := evaluateTrace(result.Trace, traceExpectation{
		requested: []string{"/sys/bin/new.txt"},
		denied:    []string{"/sys/bin/new.txt"},
	})
	notes := []string{}
	success := result.ExitCode != 0 && containsPath(result.Trace.DeniedPaths, "/sys/bin/new.txt")
	if !success {
		notes = append(notes, "mount or synthetic boundary was not denied as expected")
	}
	return ScenarioReport{
		Name:              "mount_boundary_relative_path",
		Category:          "mount_synthetic_capability_boundaries",
		Success:           success,
		SessionScoped:     false,
		AsyncCandidate:    true,
		PatchWorkflow:     false,
		DurationMS:        time.Since(start).Milliseconds(),
		TraceChecksPassed: tracePassed,
		TraceChecksTotal:  traceTotal,
		TraceCompleteness: ratio(tracePassed, traceTotal),
		Notes:             notes,
	}, nil
}

func runCommandNamespaceScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)

	stack, err := runtimeengine.New(runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullPolicy(),
		ExternalCallbacks: fs.ExternalCallbacks{
			ListExternalCommands: func(ctx context.Context) ([]contract.ExternalCommand, error) {
				return []contract.ExternalCommand{{Name: "report_tool", Summary: "report tool manual"}}, nil
			},
			RunExternalCommand: func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
				if req.Command == "report_tool" {
					return contract.ExternalCommandResult{Stdout: "report ok", Stderr: "report warning", ExitCode: 0}, nil
				}
				return contract.ExternalCommandResult{}, contract.ErrUnsupported
			},
			ReadExternalManual: func(ctx context.Context, command string) (string, error) {
				if command == "report_tool" {
					return "report tool manual", nil
				}
				return "", contract.ErrUnsupported
			},
		},
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	whichResult := stack.ExecuteResult(context.Background(), "cd /sys/bin; which ./cat")
	manResult := stack.ExecuteResult(context.Background(), "cd /sys/bin; man ./cat")
	dispatchResult := stack.ExecuteResult(context.Background(), "cd /bin; ./report_tool")
	actionableError := stack.ExecuteResult(context.Background(), "cd /task_outputs; man ./missing.txt")

	passed, total := 0, 4
	if whichResult.ExitCode == 0 && strings.TrimSpace(whichResult.Stdout) == "/sys/bin/cat" {
		passed++
	}
	if manResult.ExitCode == 0 && strings.Contains(manResult.Stdout, "cat [-n] PATH") {
		passed++
	}
	if dispatchResult.ExitCode == 0 && strings.Contains(dispatchResult.Stdout, "report ok") {
		passed++
	}
	if actionableError.ExitCode != 0 && strings.Contains(actionableError.Stdout, "not a command path") {
		passed++
	}

	notes := []string{}
	success := passed == total
	if !success {
		notes = append(notes, "command namespace normalization or actionable path-like error handling regressed")
	}
	return ScenarioReport{
		Name:              "command_namespace_consistency",
		Category:          "command_namespace_consistency",
		Success:           success,
		SessionScoped:     false,
		AsyncCandidate:    true,
		PatchWorkflow:     false,
		DurationMS:        time.Since(start).Milliseconds(),
		TraceChecksPassed: passed,
		TraceChecksTotal:  total,
		TraceCompleteness: ratio(passed, total),
		Notes:             notes,
	}, nil
}

func runTracePlanningScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)

	stack, err := runtimeengine.New(runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullPolicy(),
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	result := stack.ExecuteResult(context.Background(), "echo hello > /task_outputs/out.txt; cat /task_outputs/out.txt")
	tracePassed, traceTotal := evaluateTrace(result.Trace, traceExpectation{
		requested:    []string{"/task_outputs/out.txt"},
		written:      []string{"/task_outputs/out.txt"},
		read:         []string{"/task_outputs/out.txt"},
		bytesRead:    func(v int) bool { return v > 0 },
		bytesWritten: func(v int) bool { return v > 0 },
	})
	notes := []string{}
	success := result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "hello"
	if !success {
		notes = append(notes, "trace-planning scenario failed to produce readable output")
	}
	return ScenarioReport{
		Name:              "trace_consumable_planning",
		Category:          "trace_consumable_planning",
		Success:           success,
		SessionScoped:     false,
		AsyncCandidate:    true,
		PatchWorkflow:     false,
		DurationMS:        time.Since(start).Milliseconds(),
		TraceChecksPassed: tracePassed,
		TraceChecksTotal:  traceTotal,
		TraceCompleteness: ratio(tracePassed, traceTotal),
		Notes:             notes,
	}, nil
}

func runCancelTimeoutScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)
	docPath := filepath.Join(root, "knowledge_base", "doc.txt")
	if err := os.WriteFile(docPath, []byte("hello\n"), 0o644); err != nil {
		return ScenarioReport{}, err
	}

	stack, err := runtimeengine.New(runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := stack.ExecuteResult(cancelCtx, "cat /knowledge_base/doc.txt")

	timeoutCtx, timeoutCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer timeoutCancel()
	timedOut := stack.ExecuteResult(timeoutCtx, "cat /knowledge_base/doc.txt")

	passed, total := 0, 2
	if canceled.Trace.Canceled && canceled.ExitCode != 0 {
		passed++
	}
	if timedOut.Trace.TimedOut && timedOut.ExitCode != 0 {
		passed++
	}
	notes := []string{}
	success := passed == total
	if !success {
		notes = append(notes, "cancel or timeout flags were not surfaced as expected")
	}
	return ScenarioReport{
		Name:              "cancel_timeout_interruptions",
		Category:          "cancel_timeout_interruption",
		Success:           success,
		SessionScoped:     false,
		AsyncCandidate:    true,
		PatchWorkflow:     false,
		DurationMS:        time.Since(start).Milliseconds(),
		TraceChecksPassed: passed,
		TraceChecksTotal:  total,
		TraceCompleteness: ratio(passed, total),
		Notes:             notes,
	}, nil
}

func evaluateTrace(trace contract.ExecutionTrace, expected traceExpectation) (int, int) {
	passed := 0
	total := 0

	for _, path := range expected.requested {
		total++
		if containsPath(trace.RequestedPaths, path) {
			passed++
		}
	}
	for _, path := range expected.read {
		total++
		if containsPath(trace.ReadPaths, path) {
			passed++
		}
	}
	for _, path := range expected.written {
		total++
		if containsPath(trace.WrittenPaths, path) {
			passed++
		}
	}
	for _, path := range expected.edited {
		total++
		if containsPath(trace.EditedPaths, path) {
			passed++
		}
	}
	for _, path := range expected.denied {
		total++
		if containsPath(trace.DeniedPaths, path) {
			passed++
		}
	}
	if expected.bytesRead != nil {
		total++
		if expected.bytesRead(trace.BytesRead) {
			passed++
		}
	}
	if expected.bytesWritten != nil {
		total++
		if expected.bytesWritten(trace.BytesWritten) {
			passed++
		}
	}
	if expected.canceled != nil {
		total++
		if trace.Canceled == *expected.canceled {
			passed++
		}
	}
	if expected.timedOut != nil {
		total++
		if trace.TimedOut == *expected.timedOut {
			passed++
		}
	}
	return passed, total
}

func containsPath(paths []string, target string) bool {
	for _, pathValue := range paths {
		if pathValue == target {
			return true
		}
	}
	return false
}

func ratio(passed int, total int) float64 {
	if total == 0 {
		return 1.0
	}
	return float64(passed) / float64(total)
}

func mustTempHostRoot() string {
	root, err := os.MkdirTemp("", "simsh-bench-")
	if err != nil {
		panic(err)
	}
	dirs := []string{
		filepath.Join(root, "task_outputs"),
		filepath.Join(root, "temp_work"),
		filepath.Join(root, "knowledge_base"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
	}
	return root
}

func newFullSessionManager() *runtimeengine.SessionManager {
	return runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{})
}

func fullPolicy() contract.ExecutionPolicy {
	return contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxWriteBytes:    1 << 20,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}
}

func marshalReport(report SuiteReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

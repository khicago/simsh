package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	referenceadapter "github.com/khicago/simsh/pkg/adapter/reference"
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
	Name                  string   `json:"name"`
	Category              string   `json:"category"`
	Success               bool     `json:"success"`
	SessionScoped         bool     `json:"session_scoped"`
	AsyncCandidate        bool     `json:"async_candidate"`
	PatchWorkflow         bool     `json:"patch_workflow"`
	DurationMS            int64    `json:"duration_ms"`
	TraceChecksPassed     int      `json:"trace_checks_passed"`
	TraceChecksTotal      int      `json:"trace_checks_total"`
	TraceCompleteness     *float64 `json:"trace_completeness,omitempty"`
	AssertionChecksPassed int      `json:"assertion_checks_passed,omitempty"`
	AssertionChecksTotal  int      `json:"assertion_checks_total,omitempty"`
	AssertionCompleteness *float64 `json:"assertion_completeness,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

type projectionRecordView struct {
	Path      string `json:"path"`
	Source    string `json:"source"`
	Freshness string `json:"freshness"`
}

type projectionIndexView struct {
	Documents []projectionRecordView `json:"documents,omitempty"`
	Resources []projectionRecordView `json:"resources,omitempty"`
}

type workflowViewRecord struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	StatusSource string   `json:"status_source,omitempty"`
	StatusReason string   `json:"status_reason,omitempty"`
	Evidence     []string `json:"evidence,omitempty"`
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
		runAdapterProjectionScenario,
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
	assertionPassed, assertionTotal := countChecks(
		setup.Result.ExitCode == 0,
		pwdResult.Result.ExitCode == 0,
		strings.TrimSpace(pwdResult.Result.Stdout) == "/task_outputs/project",
		readResult.Result.ExitCode == 0,
		strings.TrimSpace(readResult.Result.Stdout) == "hello",
		readResult.Session.State.WorkingDir == "/task_outputs/project",
	)
	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "session-relative navigation or cwd persistence failed")
	}
	duration := time.Since(start).Milliseconds()
	return ScenarioReport{
		Name:                  "relative_navigation_session",
		Category:              "relative_path_navigation",
		Success:               success,
		SessionScoped:         true,
		AsyncCandidate:        false,
		PatchWorkflow:         false,
		DurationMS:            duration,
		TraceChecksPassed:     tracePassed,
		TraceChecksTotal:      traceTotal,
		TraceCompleteness:     ratioPtr(tracePassed, traceTotal),
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
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
	assertionPassed, assertionTotal := countChecks(
		result.ExitCode == 0,
		strings.TrimSpace(result.Stdout) == "beta",
	)
	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "inspect/edit/write loop did not produce final reviewable output")
	}
	return ScenarioReport{
		Name:                  "inspect_edit_write_loop",
		Category:              "file_inspect_edit_write_loops",
		Success:               success,
		SessionScoped:         true,
		AsyncCandidate:        true,
		PatchWorkflow:         true,
		DurationMS:            time.Since(start).Milliseconds(),
		TraceChecksPassed:     tracePassed,
		TraceChecksTotal:      traceTotal,
		TraceCompleteness:     ratioPtr(tracePassed, traceTotal),
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
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
	assertionPassed, assertionTotal := countChecks(result.ExitCode != 0)
	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "mount or synthetic boundary was not denied as expected")
	}
	return ScenarioReport{
		Name:                  "mount_boundary_relative_path",
		Category:              "mount_synthetic_capability_boundaries",
		Success:               success,
		SessionScoped:         false,
		AsyncCandidate:        true,
		PatchWorkflow:         false,
		DurationMS:            time.Since(start).Milliseconds(),
		TraceChecksPassed:     tracePassed,
		TraceChecksTotal:      traceTotal,
		TraceCompleteness:     ratioPtr(tracePassed, traceTotal),
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
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

	assertionPassed, assertionTotal := countChecks(
		whichResult.ExitCode == 0 && strings.TrimSpace(whichResult.Stdout) == "/sys/bin/cat",
		manResult.ExitCode == 0 && strings.Contains(manResult.Stdout, "cat [-n] PATH"),
		dispatchResult.ExitCode == 0 && strings.Contains(dispatchResult.Stdout, "report ok"),
		actionableError.ExitCode != 0 && strings.Contains(actionableError.Stdout, "not a command path"),
	)

	notes := []string{}
	success := scenarioSucceeded(0, 0, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "command namespace normalization or actionable path-like error handling regressed")
	}
	return ScenarioReport{
		Name:                  "command_namespace_consistency",
		Category:              "command_namespace_consistency",
		Success:               success,
		SessionScoped:         false,
		AsyncCandidate:        true,
		PatchWorkflow:         false,
		DurationMS:            time.Since(start).Milliseconds(),
		TraceChecksPassed:     0,
		TraceChecksTotal:      0,
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
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
	assertionPassed, assertionTotal := countChecks(
		result.ExitCode == 0,
		strings.TrimSpace(result.Stdout) == "hello",
	)
	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "trace-planning scenario failed to produce readable output")
	}
	return ScenarioReport{
		Name:                  "trace_consumable_planning",
		Category:              "trace_consumable_planning",
		Success:               success,
		SessionScoped:         false,
		AsyncCandidate:        true,
		PatchWorkflow:         false,
		DurationMS:            time.Since(start).Milliseconds(),
		TraceChecksPassed:     tracePassed,
		TraceChecksTotal:      traceTotal,
		TraceCompleteness:     ratioPtr(tracePassed, traceTotal),
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
	}, nil
}

func runAdapterProjectionScenario() (ScenarioReport, error) {
	root := mustTempHostRoot()
	defer os.RemoveAll(root)

	adapter := referenceadapter.New(referenceadapter.Options{
		Documents: map[string]string{
			"guide.md": "# Guide\nhello\n",
		},
		DocumentMetadata: map[string]referenceadapter.ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: "snapshot"},
		},
		Resources: map[string]string{
			"checklists/plan.json": "{\"steps\":[\"read\",\"write\"]}\n",
		},
		ResourceMetadata: map[string]referenceadapter.ProjectionMetadata{
			"checklists/plan.json": {Source: "workflow_catalog", Freshness: "live"},
		},
		Workflows: []referenceadapter.WorkflowSpec{
			{
				ID:              "draft-plan",
				Title:           "Draft plan",
				Summary:         "Read the planning checklist and write the first plan draft.",
				ResourcePaths:   []string{"/resources/checklists/plan.json"},
				ExpectedOutputs: []string{"/task_outputs/plan.txt"},
			},
		},
	})
	manager := newFullSessionManager()
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: root,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		return ScenarioReport{}, err
	}

	start := time.Now()
	readGuide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	readResource, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	projectionsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	initialProjectionView, err := decodeProjectionIndex(projectionsView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	workflowsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	writeOutput, err := manager.Execute(context.Background(), session.SessionID, "echo plan > /task_outputs/plan.txt", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	deniedWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	checkpoint, err := manager.Checkpoint(context.Background(), session.SessionID)
	if err != nil {
		return ScenarioReport{}, err
	}
	adapter.InvalidateDocument("guide.md")
	adapter.InvalidateResource("checklists/plan.json")
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	resumed, err := manager.Resume(context.Background(), session.SessionID)
	if err != nil {
		return ScenarioReport{}, err
	}
	staleProjectionsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	staleProjectionView, err := decodeProjectionIndex(staleProjectionsView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	staleSummaryView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	adapter.RefreshDocument("guide.md", "# Guide\nhello refreshed\n", referenceadapter.ProjectionMetadata{})
	adapter.RefreshResource("checklists/plan.json", "{\"steps\":[\"refresh\"]}\n", referenceadapter.ProjectionMetadata{})
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	refreshedGuide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	refreshedResource, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	refreshedProjectionsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	refreshedProjectionView, err := decodeProjectionIndex(refreshedProjectionsView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	adapter.SetWorkflowStatus("draft-plan", "blocked", "awaiting review")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	overrideWorkflowsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	overrideWorkflowState, err := decodeWorkflowViews(overrideWorkflowsView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	adapter.ClearWorkflowStatus("draft-plan")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	traceWorkflowsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	traceWorkflowState, err := decodeWorkflowViews(traceWorkflowsView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	memoryView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/observations.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	summaryView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}

	readTracePassed, readTraceTotal := evaluateTrace(readGuide.Result.Trace, traceExpectation{
		requested: []string{"/knowledge_base/reference/guide.md"},
		read:      []string{"/knowledge_base/reference/guide.md"},
	})
	resourceTracePassed, resourceTraceTotal := evaluateTrace(readResource.Result.Trace, traceExpectation{
		requested: []string{"/resources/checklists/plan.json"},
		read:      []string{"/resources/checklists/plan.json"},
	})
	writeTracePassed, writeTraceTotal := evaluateTrace(writeOutput.Result.Trace, traceExpectation{
		requested: []string{"/task_outputs/plan.txt"},
		written:   []string{"/task_outputs/plan.txt"},
	})
	denyTracePassed, denyTraceTotal := evaluateTrace(deniedWrite.Result.Trace, traceExpectation{
		requested: []string{"/knowledge_base/reference/guide.md"},
		denied:    []string{"/knowledge_base/reference/guide.md"},
	})
	tracePassed := readTracePassed + resourceTracePassed + writeTracePassed + denyTracePassed
	traceTotal := readTraceTotal + resourceTraceTotal + writeTraceTotal + denyTraceTotal
	initialGuideProjection, initialGuideOK := findProjectionRecord(initialProjectionView.Documents, "/knowledge_base/reference/guide.md")
	initialResourceProjection, initialResourceOK := findProjectionRecord(initialProjectionView.Resources, "/resources/checklists/plan.json")
	staleGuideProjection, staleGuideOK := findProjectionRecord(staleProjectionView.Documents, "/knowledge_base/reference/guide.md")
	staleResourceProjection, staleResourceOK := findProjectionRecord(staleProjectionView.Resources, "/resources/checklists/plan.json")
	refreshedGuideProjection, refreshedGuideOK := findProjectionRecord(refreshedProjectionView.Documents, "/knowledge_base/reference/guide.md")
	refreshedResourceProjection, refreshedResourceOK := findProjectionRecord(refreshedProjectionView.Resources, "/resources/checklists/plan.json")
	overrideWorkflow, overrideWorkflowOK := findWorkflowView(overrideWorkflowState, "draft-plan")
	traceWorkflow, traceWorkflowOK := findWorkflowView(traceWorkflowState, "draft-plan")
	assertionPassed, assertionTotal := countChecks(
		strings.Contains(readGuide.Result.Stdout, "# Guide"),
		strings.Contains(readResource.Result.Stdout, "\"steps\""),
		initialGuideOK && initialGuideProjection.Source == "knowledge_sync" && initialGuideProjection.Freshness == "snapshot",
		initialResourceOK && initialResourceProjection.Source == "workflow_catalog" && initialResourceProjection.Freshness == "live",
		strings.Contains(workflowsView.Result.Stdout, "[in_progress] Draft plan (draft-plan)"),
		writeOutput.Result.ExitCode == 0,
		deniedWrite.Result.ExitCode != 0,
		len(checkpoint.State.Opaque[adapter.AdapterID()]) > 0,
		len(resumed.State.Opaque[adapter.AdapterID()]) > 0,
		staleGuideOK && staleGuideProjection.Source == "knowledge_sync" && staleGuideProjection.Freshness == "stale",
		staleResourceOK && staleResourceProjection.Source == "workflow_catalog" && staleResourceProjection.Freshness == "stale",
		strings.Contains(staleSummaryView.Result.Stdout, "- stale: 2"),
		strings.Contains(refreshedGuide.Result.Stdout, "hello refreshed"),
		strings.Contains(refreshedResource.Result.Stdout, "\"refresh\""),
		refreshedGuideOK && refreshedGuideProjection.Source == "knowledge_sync" && refreshedGuideProjection.Freshness == "live",
		refreshedResourceOK && refreshedResourceProjection.Source == "workflow_catalog" && refreshedResourceProjection.Freshness == "live",
		overrideWorkflowOK &&
			overrideWorkflow.Status == "blocked" &&
			overrideWorkflow.StatusSource == "control_plane" &&
			overrideWorkflow.StatusReason == "awaiting review" &&
			containsPath(overrideWorkflow.Evidence, "/task_outputs/plan.txt"),
		traceWorkflowOK &&
			traceWorkflow.Status == "completed" &&
			traceWorkflow.StatusSource == "trace" &&
			traceWorkflow.StatusReason == "",
		strings.Contains(memoryView.Result.Stdout, "read-ref:/knowledge_base/reference/guide.md") &&
			strings.Contains(memoryView.Result.Stdout, "read-resource:/resources/checklists/plan.json") &&
			strings.Contains(memoryView.Result.Stdout, "wrote:/task_outputs/plan.txt") &&
			strings.Contains(memoryView.Result.Stdout, "denied:/knowledge_base/reference/guide.md"),
		strings.Contains(summaryView.Result.Stdout, "resource_reads: 1") &&
			strings.Contains(summaryView.Result.Stdout, "written_outputs: 1"),
	)

	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "adapter-backed projection or managed /memory lifecycle regressed")
	}
	return ScenarioReport{
		Name:                  "adapter_projection_memory_lifecycle",
		Category:              "adapter_backed_projection_validation",
		Success:               success,
		SessionScoped:         true,
		AsyncCandidate:        true,
		PatchWorkflow:         false,
		DurationMS:            time.Since(start).Milliseconds(),
		TraceChecksPassed:     tracePassed,
		TraceChecksTotal:      traceTotal,
		TraceCompleteness:     ratioPtr(tracePassed, traceTotal),
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
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

	tracePassed, traceTotal := evaluateTrace(canceled.Trace, traceExpectation{canceled: boolPtr(true)})
	tracePassed2, traceTotal2 := evaluateTrace(timedOut.Trace, traceExpectation{timedOut: boolPtr(true)})
	tracePassed += tracePassed2
	traceTotal += traceTotal2
	assertionPassed, assertionTotal := countChecks(
		canceled.ExitCode != 0,
		timedOut.ExitCode != 0,
	)
	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "cancel or timeout flags were not surfaced as expected")
	}
	return ScenarioReport{
		Name:                  "cancel_timeout_interruptions",
		Category:              "cancel_timeout_interruption",
		Success:               success,
		SessionScoped:         false,
		AsyncCandidate:        true,
		PatchWorkflow:         false,
		DurationMS:            time.Since(start).Milliseconds(),
		TraceChecksPassed:     tracePassed,
		TraceChecksTotal:      traceTotal,
		TraceCompleteness:     ratioPtr(tracePassed, traceTotal),
		AssertionChecksPassed: assertionPassed,
		AssertionChecksTotal:  assertionTotal,
		AssertionCompleteness: ratioPtr(assertionPassed, assertionTotal),
		Notes:                 notes,
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

func ratioPtr(passed int, total int) *float64 {
	if total == 0 {
		return nil
	}
	value := ratio(passed, total)
	return &value
}

func countChecks(checks ...bool) (int, int) {
	passed := 0
	for _, ok := range checks {
		if ok {
			passed++
		}
	}
	return passed, len(checks)
}

func scenarioSucceeded(tracePassed int, traceTotal int, assertionPassed int, assertionTotal int) bool {
	if assertionTotal > 0 && assertionPassed != assertionTotal {
		return false
	}
	if traceTotal > 0 && tracePassed != traceTotal {
		return false
	}
	return true
}

func decodeProjectionIndex(raw string) (projectionIndexView, error) {
	var view projectionIndexView
	err := json.Unmarshal([]byte(raw), &view)
	return view, err
}

func findProjectionRecord(records []projectionRecordView, target string) (projectionRecordView, bool) {
	for _, record := range records {
		if record.Path == target {
			return record, true
		}
	}
	return projectionRecordView{}, false
}

func decodeWorkflowViews(raw string) ([]workflowViewRecord, error) {
	var workflows []workflowViewRecord
	err := json.Unmarshal([]byte(raw), &workflows)
	return workflows, err
}

func findWorkflowView(workflows []workflowViewRecord, id string) (workflowViewRecord, bool) {
	for _, workflow := range workflows {
		if workflow.ID == id {
			return workflow, true
		}
	}
	return workflowViewRecord{}, false
}

func boolPtr(value bool) *bool {
	return &value
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
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

type projectionFailureView struct {
	Code string `json:"code,omitempty"`
}

type projectionMaterializationView struct {
	State   string                 `json:"state"`
	Reason  string                 `json:"reason,omitempty"`
	Failure *projectionFailureView `json:"failure,omitempty"`
}

type projectionRecordView struct {
	Path            string                         `json:"path"`
	Source          string                         `json:"source"`
	Freshness       string                         `json:"freshness"`
	Materialization *projectionMaterializationView `json:"materialization,omitempty"`
	Eligibility     *skillEligibilityRecord        `json:"eligibility,omitempty"`
	Precedence      *skillPrecedenceRecord         `json:"precedence,omitempty"`
	Selection       *skillSelectionRecord          `json:"selection,omitempty"`
	Selected        bool                           `json:"selected,omitempty"`
}

type skillEligibilityRecord struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type skillPrecedenceRecord struct {
	Tier string `json:"tier"`
	Rank int    `json:"rank"`
}

type skillSelectionRecord struct {
	Scope      string `json:"scope,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Reason     string `json:"reason,omitempty"`
	WinnerPath string `json:"winner_path,omitempty"`
}

type skillAuditRecord struct {
	Seq                   int    `json:"seq"`
	Op                    string `json:"op"`
	Path                  string `json:"path"`
	SelectionScope        string `json:"selection_scope,omitempty"`
	Result                string `json:"result"`
	Visibility            string `json:"visibility"`
	VisibleAfter          string `json:"visible_after"`
	VisibleFromGeneration int    `json:"visible_from_generation"`
	SelectedBefore        bool   `json:"selected_before"`
	SelectedAfter         bool   `json:"selected_after"`
	WinnerBefore          string `json:"winner_before,omitempty"`
	WinnerAfter           string `json:"winner_after,omitempty"`
	ReasonAfter           string `json:"reason_after,omitempty"`
}

type projectionIndexView struct {
	Documents []projectionRecordView `json:"documents,omitempty"`
	Resources []projectionRecordView `json:"resources,omitempty"`
	Skills    []projectionRecordView `json:"skills,omitempty"`
}

type workflowViewRecord struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	StatusSource string   `json:"status_source,omitempty"`
	StatusReason string   `json:"status_reason,omitempty"`
	Evidence     []string `json:"evidence,omitempty"`
}

type curatedMemorySnapshot struct {
	EntryCount  int
	SourcePaths []string
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

type namedCheck struct {
	name string
	ok   bool
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
		Skills: map[string]string{
			"planning/draft-plan": "# Draft-plan skill\n",
			"planning/alternate":  "# Alternate planning skill\n",
			"planning/fallback":   "# Fallback planning skill\n",
		},
		SkillMetadata: map[string]referenceadapter.SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "planning/default",
				Eligibility: referenceadapter.SkillEligibility{
					State: "eligible",
				},
				Precedence: referenceadapter.SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: false,
			},
			"planning/alternate": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "planning/default",
				Eligibility: referenceadapter.SkillEligibility{
					State: "eligible",
				},
				Precedence: referenceadapter.SkillPrecedence{
					Tier: "workspace",
					Rank: 5,
				},
				Selected: true,
			},
			"planning/fallback": {
				Source:         "bundled_catalog",
				Freshness:      "snapshot",
				SelectionScope: "planning/default",
				Eligibility: referenceadapter.SkillEligibility{
					State:  "ineligible",
					Reason: "missing_env:PLAN_FALLBACK_TOKEN",
				},
				Precedence: referenceadapter.SkillPrecedence{
					Tier: "bundled",
					Rank: 90,
				},
			},
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
	readSkill, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/planning/draft-plan/SKILL.md", contract.ExecutionPolicy{})
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
	skillsIndexView, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	initialSkillIndex, err := decodeProjectionRecordsView(skillsIndexView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	workflowsView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	workflowStateView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	workflowState, err := decodeWorkflowViews(workflowStateView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	writeOutput, err := manager.Execute(context.Background(), session.SessionID, "echo plan > /task_outputs/plan.txt", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	adapter.UpsertCuratedEntry(referenceadapter.CuratedEntry{
		ID:          "plan-context",
		Title:       "Plan Context",
		Summary:     "Curated evidence for the current planning workflow.",
		SourcePaths: []string{"/knowledge_base/reference/guide.md", "/resources/checklists/plan.json", "/task_outputs/plan.txt", "/skills/planning/draft-plan/SKILL.md"},
	})
	deniedWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	deniedSkillWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /skills/planning/draft-plan/SKILL.md", contract.ExecutionPolicy{})
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
	adapter.UpsertSkill("planning/live-hotfix", "# Live hotfix skill\n", referenceadapter.SkillMetadata{
		Source:         "control_plane",
		Freshness:      "updated",
		SelectionScope: "planning/default",
		Eligibility:    referenceadapter.SkillEligibility{State: "eligible"},
		Precedence:     referenceadapter.SkillPrecedence{Tier: "workspace", Rank: 0},
	})
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillProjectionView, err := decodeProjectionIndex(controlPlaneSkillView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillAuditView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills_audit.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillAudit, err := decodeSkillAudit(controlPlaneSkillAuditView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillContent, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/planning/live-hotfix/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	adapter.UpdateSkill("planning/live-hotfix", "# Live hotfix skill\nUpdated guidance.\n")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillUpdatedView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillUpdatedProjectionView, err := decodeProjectionIndex(controlPlaneSkillUpdatedView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillUpdatedAuditView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills_audit.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneSkillUpdatedAudit, err := decodeSkillAudit(controlPlaneSkillUpdatedAuditView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	controlPlaneUpdatedSkillContent, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/planning/live-hotfix/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	adapter.RemoveSkill("planning/live-hotfix")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		return ScenarioReport{}, err
	}
	postRemoveSkillView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	postRemoveSkillProjectionView, err := decodeProjectionIndex(postRemoveSkillView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	postRemoveSkillAuditView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills_audit.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	postRemoveSkillAudit, err := decodeSkillAudit(postRemoveSkillAuditView.Result.Stdout)
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
	summaryText := summaryView.Result.Stdout
	lastSummaryValue, lastSummaryOk := summaryLineValue(summaryText, "last_control_plane_event")
	curatedViewPath := "/memory/curated.json"
	curatedView, err := manager.Execute(context.Background(), session.SessionID, "cat "+curatedViewPath, contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	curatedSnapshot, err := decodeCuratedMemorySnapshot(curatedView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	skillsAuditView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills_audit.json", contract.ExecutionPolicy{})
	if err != nil {
		return ScenarioReport{}, err
	}
	if skillsAuditView.Result.ExitCode != 0 {
		return ScenarioReport{}, fmt.Errorf("skills audit command failed: %+v", skillsAuditView.Result)
	}
	controlPlaneAuditHistory, err := decodeSkillAudit(skillsAuditView.Result.Stdout)
	if err != nil {
		return ScenarioReport{}, err
	}
	statusSnapshot, err := manager.Get(session.SessionID)
	if err != nil {
		return ScenarioReport{}, err
	}
	rawState, ok := statusSnapshot.State.Opaque[adapter.AdapterID()]
	if !ok {
		return ScenarioReport{}, fmt.Errorf("missing adapter state for %q", adapter.AdapterID())
	}
	auditState, err := decodeSessionStateSummary(rawState)
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
	denySkillTracePassed, denySkillTraceTotal := evaluateTrace(deniedSkillWrite.Result.Trace, traceExpectation{
		requested: []string{"/skills/planning/draft-plan/SKILL.md"},
		denied:    []string{"/skills/planning/draft-plan/SKILL.md"},
	})
	tracePassed := readTracePassed + resourceTracePassed + writeTracePassed + denyTracePassed + denySkillTracePassed
	traceTotal := readTraceTotal + resourceTraceTotal + writeTraceTotal + denyTraceTotal + denySkillTraceTotal
	initialGuideProjection, initialGuideOK := findProjectionRecord(initialProjectionView.Documents, "/knowledge_base/reference/guide.md")
	initialResourceProjection, initialResourceOK := findProjectionRecord(initialProjectionView.Resources, "/resources/checklists/plan.json")
	initialSkillProjection, initialSkillOK := findProjectionRecord(initialProjectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	initialSkillAlternate, initialSkillAlternateOK := findProjectionRecord(initialProjectionView.Skills, "/skills/planning/alternate/SKILL.md")
	initialSkillFallback, initialSkillFallbackOK := findProjectionRecord(initialProjectionView.Skills, "/skills/planning/fallback/SKILL.md")
	skillIndexDraft, skillIndexDraftOK := findProjectionRecord(initialSkillIndex, "/skills/planning/draft-plan/SKILL.md")
	staleGuideProjection, staleGuideOK := findProjectionRecord(staleProjectionView.Documents, "/knowledge_base/reference/guide.md")
	staleResourceProjection, staleResourceOK := findProjectionRecord(staleProjectionView.Resources, "/resources/checklists/plan.json")
	staleSkillProjection, staleSkillOK := findProjectionRecord(staleProjectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	refreshedGuideProjection, refreshedGuideOK := findProjectionRecord(refreshedProjectionView.Documents, "/knowledge_base/reference/guide.md")
	refreshedResourceProjection, refreshedResourceOK := findProjectionRecord(refreshedProjectionView.Resources, "/resources/checklists/plan.json")
	refreshedSkillProjection, refreshedSkillOK := findProjectionRecord(refreshedProjectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	controlPlaneHotfixProjection, controlPlaneHotfixOK := findProjectionRecord(controlPlaneSkillProjectionView.Skills, "/skills/planning/live-hotfix/SKILL.md")
	controlPlaneDraftProjection, controlPlaneDraftOK := findProjectionRecord(controlPlaneSkillProjectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	controlPlaneUpdatedProjection, controlPlaneUpdatedOK := findProjectionRecord(controlPlaneSkillUpdatedProjectionView.Skills, "/skills/planning/live-hotfix/SKILL.md")
	postRemoveDraftProjection, postRemoveDraftOK := findProjectionRecord(postRemoveSkillProjectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	_, postRemoveHotfixOK := findProjectionRecord(postRemoveSkillProjectionView.Skills, "/skills/planning/live-hotfix/SKILL.md")
	overrideWorkflow, overrideWorkflowOK := findWorkflowView(overrideWorkflowState, "draft-plan")
	traceWorkflow, traceWorkflowOK := findWorkflowView(traceWorkflowState, "draft-plan")
	workflowSkillBacked, workflowSkillBackedOK := findWorkflowView(workflowState, "draft-plan")
	assertionPassed, assertionTotal, failedChecks := countNamedChecks(
		namedCheck{name: "read_guide_content", ok: strings.Contains(readGuide.Result.Stdout, "# Guide")},
		namedCheck{name: "read_resource_content", ok: strings.Contains(readResource.Result.Stdout, "\"steps\"")},
		namedCheck{name: "read_skill_content", ok: strings.Contains(readSkill.Result.Stdout, "Draft-plan skill")},
		namedCheck{name: "initial_document_projection", ok: initialGuideOK && initialGuideProjection.Source == "knowledge_sync" && initialGuideProjection.Freshness == "snapshot"},
		namedCheck{name: "initial_resource_projection", ok: initialResourceOK && initialResourceProjection.Source == "workflow_catalog" && initialResourceProjection.Freshness == "live"},
		namedCheck{name: "initial_document_materialization_state", ok: initialGuideOK && projectionHasMaterializationState(initialGuideProjection)},
		namedCheck{name: "initial_resource_materialization_state", ok: initialResourceOK && projectionHasMaterializationState(initialResourceProjection)},
		namedCheck{name: "initial_skill_projection", ok: initialSkillOK &&
			initialSkillProjection.Source == "workspace_catalog" &&
			initialSkillProjection.Freshness == "live" &&
			skillEligibilityState(initialSkillProjection) == "eligible" &&
			skillPrecedenceTier(initialSkillProjection) == "workspace" &&
			skillPrecedenceRank(initialSkillProjection) == 1 &&
			skillSelected(initialSkillProjection)},
		namedCheck{name: "initial_skill_selection_provenance", ok: initialSkillOK &&
			skillSelectionScope(initialSkillProjection) != "" &&
			skillSelectionMode(initialSkillProjection) != ""},
		namedCheck{name: "initial_skill_materialization_state", ok: initialSkillOK && projectionHasMaterializationState(initialSkillProjection)},
		namedCheck{name: "alternate_skill_projection", ok: initialSkillAlternateOK &&
			initialSkillAlternate.Source == "workspace_catalog" &&
			initialSkillAlternate.Freshness == "live" &&
			skillEligibilityState(initialSkillAlternate) == "eligible" &&
			skillPrecedenceTier(initialSkillAlternate) == "workspace" &&
			skillPrecedenceRank(initialSkillAlternate) == 5 &&
			!skillSelected(initialSkillAlternate)},
		namedCheck{name: "alternate_skill_selection_provenance", ok: initialSkillAlternateOK &&
			skillSelectionScope(initialSkillAlternate) != "" &&
			skillSelectionMode(initialSkillAlternate) != "" &&
			skillSelectionReason(initialSkillAlternate) != "" &&
			skillSelectionScope(initialSkillAlternate) == skillSelectionScope(initialSkillProjection)},
		namedCheck{name: "alternate_skill_materialization_state", ok: initialSkillAlternateOK && projectionHasMaterializationState(initialSkillAlternate)},
		namedCheck{name: "fallback_skill_projection", ok: initialSkillFallbackOK &&
			initialSkillFallback.Source == "bundled_catalog" &&
			initialSkillFallback.Freshness == "snapshot" &&
			skillEligibilityState(initialSkillFallback) == "ineligible" &&
			skillEligibilityReason(initialSkillFallback) == "missing_env:PLAN_FALLBACK_TOKEN" &&
			skillPrecedenceTier(initialSkillFallback) == "bundled" &&
			skillPrecedenceRank(initialSkillFallback) == 90 &&
			!skillSelected(initialSkillFallback)},
		namedCheck{name: "fallback_skill_selection_provenance", ok: initialSkillFallbackOK &&
			skillSelectionScope(initialSkillFallback) != "" &&
			skillSelectionMode(initialSkillFallback) != "" &&
			skillSelectionReason(initialSkillFallback) != ""},
		namedCheck{name: "fallback_skill_materialization_state", ok: initialSkillFallbackOK && projectionHasMaterializationState(initialSkillFallback)},
		namedCheck{name: "skills_index_projection", ok: skillIndexDraftOK &&
			skillIndexDraft.Source == "workspace_catalog" &&
			skillEligibilityState(skillIndexDraft) == "eligible" &&
			skillPrecedenceTier(skillIndexDraft) == "workspace" &&
			skillSelected(skillIndexDraft) &&
			skillSelectionScope(skillIndexDraft) != "" &&
			skillSelectionMode(skillIndexDraft) != ""},
		namedCheck{name: "workflows_markdown_state", ok: strings.Contains(workflowsView.Result.Stdout, "[in_progress] Draft plan (draft-plan)")},
		namedCheck{name: "workflow_skill_backed_state", ok: workflowSkillBackedOK &&
			(workflowSkillBacked.Status == "pending" || workflowSkillBacked.Status == "in_progress") &&
			workflowSkillBacked.StatusSource == "trace" &&
			skillEligibilityState(initialSkillProjection) == "eligible"},
		namedCheck{name: "write_output_exit_code", ok: writeOutput.Result.ExitCode == 0},
		namedCheck{name: "deny_reference_write_exit_code", ok: deniedWrite.Result.ExitCode != 0},
		namedCheck{name: "deny_skill_write_exit_code", ok: deniedSkillWrite.Result.ExitCode != 0},
		namedCheck{name: "checkpoint_has_opaque_state", ok: len(checkpoint.State.Opaque[adapter.AdapterID()]) > 0},
		namedCheck{name: "resume_has_opaque_state", ok: len(resumed.State.Opaque[adapter.AdapterID()]) > 0},
		namedCheck{name: "stale_document_projection", ok: staleGuideOK && staleGuideProjection.Source == "knowledge_sync" && staleGuideProjection.Freshness == "stale"},
		namedCheck{name: "stale_resource_projection", ok: staleResourceOK && staleResourceProjection.Source == "workflow_catalog" && staleResourceProjection.Freshness == "stale"},
		namedCheck{name: "stale_skill_projection", ok: staleSkillOK && staleSkillProjection.Source == "workspace_catalog" && staleSkillProjection.Freshness == "live" && skillEligibilityState(staleSkillProjection) == "eligible"},
		namedCheck{name: "stale_document_materialization_partial_or_error", ok: staleGuideOK &&
			projectionMaterializationStateIn(staleGuideProjection, "partial", "error") &&
			projectionHasMaterializationFailureDetail(staleGuideProjection)},
		namedCheck{name: "stale_resource_materialization_partial_or_error", ok: staleResourceOK &&
			projectionMaterializationStateIn(staleResourceProjection, "partial", "error") &&
			projectionHasMaterializationFailureDetail(staleResourceProjection)},
		namedCheck{name: "stale_skill_materialization_state", ok: staleSkillOK && projectionHasMaterializationState(staleSkillProjection)},
		namedCheck{name: "stale_summary_count", ok: strings.Contains(staleSummaryView.Result.Stdout, "- stale: 2")},
		namedCheck{name: "refreshed_guide_content", ok: strings.Contains(refreshedGuide.Result.Stdout, "hello refreshed")},
		namedCheck{name: "refreshed_resource_content", ok: strings.Contains(refreshedResource.Result.Stdout, "\"refresh\"")},
		namedCheck{name: "refreshed_document_projection", ok: refreshedGuideOK && refreshedGuideProjection.Source == "knowledge_sync" && refreshedGuideProjection.Freshness == "live"},
		namedCheck{name: "refreshed_resource_projection", ok: refreshedResourceOK && refreshedResourceProjection.Source == "workflow_catalog" && refreshedResourceProjection.Freshness == "live"},
		namedCheck{name: "refreshed_skill_projection", ok: refreshedSkillOK && refreshedSkillProjection.Source == "workspace_catalog" && refreshedSkillProjection.Freshness == "live" && skillPrecedenceRank(refreshedSkillProjection) == 1 && skillSelected(refreshedSkillProjection)},
		namedCheck{name: "refreshed_document_materialization_state", ok: refreshedGuideOK && projectionHasMaterializationState(refreshedGuideProjection)},
		namedCheck{name: "refreshed_resource_materialization_state", ok: refreshedResourceOK && projectionHasMaterializationState(refreshedResourceProjection)},
		namedCheck{name: "refreshed_skill_materialization_state", ok: refreshedSkillOK && projectionHasMaterializationState(refreshedSkillProjection)},
		namedCheck{name: "control_plane_skill_added", ok: controlPlaneHotfixOK &&
			controlPlaneHotfixProjection.Source == "control_plane" &&
			controlPlaneHotfixProjection.Freshness == "updated" &&
			skillSelected(controlPlaneHotfixProjection) &&
			skillSelectionScope(controlPlaneHotfixProjection) == "planning/default" &&
			skillSelectionMode(controlPlaneHotfixProjection) != ""},
		namedCheck{name: "control_plane_skill_added_audit", ok: len(controlPlaneSkillAudit) == 1 &&
			controlPlaneSkillAudit[0].Op == "skill_added" &&
			controlPlaneSkillAudit[0].Visibility == "visible" &&
			controlPlaneSkillAudit[0].VisibleAfter == "next_projection_rebuild" &&
			controlPlaneSkillAudit[0].WinnerAfter == "/skills/planning/live-hotfix/SKILL.md"},
		namedCheck{name: "control_plane_skill_content", ok: strings.Contains(controlPlaneSkillContent.Result.Stdout, "Live hotfix skill")},
		namedCheck{name: "control_plane_skill_updated", ok: controlPlaneUpdatedOK &&
			controlPlaneUpdatedProjection.Source == "control_plane" &&
			controlPlaneUpdatedProjection.Freshness == "updated" &&
			skillSelected(controlPlaneUpdatedProjection) &&
			strings.Contains(controlPlaneUpdatedSkillContent.Result.Stdout, "Updated guidance")},
		namedCheck{name: "control_plane_skill_updated_audit", ok: len(controlPlaneSkillUpdatedAudit) == 2 &&
			controlPlaneSkillUpdatedAudit[1].Op == "skill_updated" &&
			controlPlaneSkillUpdatedAudit[1].Visibility == "visible" &&
			controlPlaneSkillUpdatedAudit[1].SelectedBefore &&
			controlPlaneSkillUpdatedAudit[1].SelectedAfter &&
			controlPlaneSkillUpdatedAudit[1].ReasonAfter != ""},
		namedCheck{name: "control_plane_skill_reselection", ok: controlPlaneDraftOK &&
			!skillSelected(controlPlaneDraftProjection) &&
			skillSelectionReason(controlPlaneDraftProjection) != "" &&
			skillSelectionWinnerPath(controlPlaneDraftProjection) == "/skills/planning/live-hotfix/SKILL.md"},
		namedCheck{name: "control_plane_skill_removed", ok: !postRemoveHotfixOK},
		namedCheck{name: "control_plane_skill_removed_audit", ok: len(postRemoveSkillAudit) == 3 &&
			postRemoveSkillAudit[2].Op == "skill_removed" &&
			!postRemoveSkillAudit[2].SelectedAfter &&
			postRemoveSkillAudit[2].VisibleAfter == "next_projection_rebuild"},
		namedCheck{name: "control_plane_skill_fallback_restored", ok: postRemoveDraftOK &&
			skillSelected(postRemoveDraftProjection) &&
			skillSelectionScope(postRemoveDraftProjection) == "planning/default" &&
			skillSelectionWinnerPath(postRemoveDraftProjection) == ""},
		namedCheck{name: "skills_audit_history_count", ok: len(controlPlaneAuditHistory) == 3},
		namedCheck{name: "skills_audit_history_sequence", ok: len(controlPlaneAuditHistory) == 3 &&
			controlPlaneAuditHistory[0].Op == "skill_added" &&
			controlPlaneAuditHistory[1].Op == "skill_updated" &&
			controlPlaneAuditHistory[2].Op == "skill_removed"},
		namedCheck{name: "skills_audit_selection_alignment", ok: len(controlPlaneAuditHistory) == 3 &&
			controlPlaneAuditHistory[0].SelectedAfter &&
			controlPlaneAuditHistory[1].SelectedAfter &&
			!controlPlaneAuditHistory[2].SelectedAfter},
		namedCheck{name: "skills_audit_visibility_timing", ok: len(controlPlaneAuditHistory) == 3 &&
			controlPlaneAuditHistory[2].VisibleAfter == "next_projection_rebuild" &&
			controlPlaneAuditHistory[2].VisibleFromGeneration <= auditState.ProjectionGeneration},
		namedCheck{name: "skills_audit_state_alignment", ok: len(controlPlaneAuditHistory) > 0 &&
			auditState.ControlPlaneEvents == len(controlPlaneAuditHistory) &&
			auditState.LastControlPlaneKind == controlPlaneAuditHistory[len(controlPlaneAuditHistory)-1].Op},
		namedCheck{name: "memory_summary_control_plane_audit", ok: len(controlPlaneAuditHistory) > 0 &&
			summaryCountEquals(summaryText, "control_plane_events", len(controlPlaneAuditHistory)) &&
			lastSummaryOk &&
			lastSummaryValue == controlPlaneAuditHistory[len(controlPlaneAuditHistory)-1].Op+" "+controlPlaneAuditHistory[len(controlPlaneAuditHistory)-1].Path},
		namedCheck{name: "workflow_override_control_plane", ok: overrideWorkflowOK &&
			overrideWorkflow.Status == "blocked" &&
			overrideWorkflow.StatusSource == "control_plane" &&
			overrideWorkflow.StatusReason == "awaiting review" &&
			containsPath(overrideWorkflow.Evidence, "/task_outputs/plan.txt")},
		namedCheck{name: "workflow_trace_after_clear", ok: traceWorkflowOK &&
			traceWorkflow.Status == "completed" &&
			traceWorkflow.StatusSource == "trace" &&
			traceWorkflow.StatusReason == ""},
		namedCheck{name: "memory_observations", ok: strings.Contains(memoryView.Result.Stdout, "read-ref:/knowledge_base/reference/guide.md") &&
			strings.Contains(memoryView.Result.Stdout, "read-resource:/resources/checklists/plan.json") &&
			strings.Contains(memoryView.Result.Stdout, "read-skill:/skills/planning/draft-plan/SKILL.md") &&
			strings.Contains(memoryView.Result.Stdout, "wrote:/task_outputs/plan.txt") &&
			strings.Contains(memoryView.Result.Stdout, "denied:/knowledge_base/reference/guide.md") &&
			strings.Contains(memoryView.Result.Stdout, "denied:/skills/planning/draft-plan/SKILL.md")},
		namedCheck{name: "memory_summary_counts", ok: summaryCountEquals(summaryView.Result.Stdout, "resource_reads", 1) &&
			summaryCountAtLeast(summaryView.Result.Stdout, "skill_reads", 3) &&
			summaryCountEquals(summaryView.Result.Stdout, "written_outputs", 1) &&
			summaryCountEquals(summaryView.Result.Stdout, "projections.skills", 3)},
		namedCheck{name: "curated_memory_entry", ok: curatedSnapshot.EntryCount > 0 &&
			containsPath(curatedSnapshot.SourcePaths, "/knowledge_base/reference/guide.md") &&
			containsPath(curatedSnapshot.SourcePaths, "/resources/checklists/plan.json") &&
			containsPath(curatedSnapshot.SourcePaths, "/task_outputs/plan.txt") &&
			containsPath(curatedSnapshot.SourcePaths, "/skills/planning/draft-plan/SKILL.md")},
	)

	notes := []string{}
	success := scenarioSucceeded(tracePassed, traceTotal, assertionPassed, assertionTotal)
	if !success {
		notes = append(notes, "adapter-backed projection or managed /memory lifecycle regressed")
		if len(failedChecks) > 0 {
			notes = append(notes, "failed_checks: "+strings.Join(failedChecks, ", "))
		}
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

func countNamedChecks(checks ...namedCheck) (int, int, []string) {
	passed := 0
	failed := make([]string, 0)
	for _, check := range checks {
		if check.ok {
			passed++
			continue
		}
		failed = append(failed, check.name)
	}
	return passed, len(checks), failed
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

func decodeProjectionRecordsView(raw string) ([]projectionRecordView, error) {
	var records []projectionRecordView
	err := json.Unmarshal([]byte(raw), &records)
	return records, err
}

func decodeSkillAudit(raw string) ([]skillAuditRecord, error) {
	var records []skillAuditRecord
	err := json.Unmarshal([]byte(raw), &records)
	return records, err
}

func decodeCuratedMemorySnapshot(raw string) (curatedMemorySnapshot, error) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return curatedMemorySnapshot{}, err
	}
	entryCount, sourcePaths := collectCuratedEntrySources(payload)
	if entryCount == 0 || len(sourcePaths) == 0 {
		return curatedMemorySnapshot{}, os.ErrInvalid
	}
	return curatedMemorySnapshot{
		EntryCount:  entryCount,
		SourcePaths: sourcePaths,
	}, nil
}

func collectCuratedEntrySources(payload any) (int, []string) {
	unique := map[string]struct{}{}
	entryCount := 0
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			paths := sourcePathsFromCuratedRecord(typed)
			if len(paths) > 0 {
				entryCount++
				for _, pathValue := range paths {
					unique[pathValue] = struct{}{}
				}
			}
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(payload)
	paths := make([]string, 0, len(unique))
	for pathValue := range unique {
		paths = append(paths, pathValue)
	}
	sort.Strings(paths)
	return entryCount, paths
}

func sourcePathsFromCuratedRecord(record map[string]any) []string {
	keys := []string{"source_path", "source_paths", "sourcePath", "sourcePaths"}
	paths := make([]string, 0)
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		paths = append(paths, extractCuratedPaths(value)...)
	}
	return dedupePaths(paths)
}

func extractCuratedPaths(value any) []string {
	switch typed := value.(type) {
	case string:
		pathValue := strings.TrimSpace(typed)
		if strings.HasPrefix(pathValue, "/") {
			return []string{pathValue}
		}
	case []any:
		paths := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				continue
			}
			pathValue := strings.TrimSpace(text)
			if strings.HasPrefix(pathValue, "/") {
				paths = append(paths, pathValue)
			}
		}
		return dedupePaths(paths)
	}
	return nil
}

func dedupePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	unique := map[string]struct{}{}
	for _, pathValue := range paths {
		if strings.TrimSpace(pathValue) == "" {
			continue
		}
		unique[pathValue] = struct{}{}
	}
	out := make([]string, 0, len(unique))
	for pathValue := range unique {
		out = append(out, pathValue)
	}
	sort.Strings(out)
	return out
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

func skillEligibilityState(record projectionRecordView) string {
	if record.Eligibility == nil {
		return ""
	}
	return record.Eligibility.State
}

func skillEligibilityReason(record projectionRecordView) string {
	if record.Eligibility == nil {
		return ""
	}
	return record.Eligibility.Reason
}

func skillPrecedenceTier(record projectionRecordView) string {
	if record.Precedence == nil {
		return ""
	}
	return record.Precedence.Tier
}

func skillPrecedenceRank(record projectionRecordView) int {
	if record.Precedence == nil {
		return -1
	}
	return record.Precedence.Rank
}

func skillSelected(record projectionRecordView) bool {
	return record.Selected
}

func skillSelectionScope(record projectionRecordView) string {
	if record.Selection == nil {
		return ""
	}
	return strings.TrimSpace(record.Selection.Scope)
}

func skillSelectionMode(record projectionRecordView) string {
	if record.Selection == nil {
		return ""
	}
	return strings.TrimSpace(record.Selection.Mode)
}

func skillSelectionReason(record projectionRecordView) string {
	if record.Selection == nil {
		return ""
	}
	return strings.TrimSpace(record.Selection.Reason)
}

func skillSelectionWinnerPath(record projectionRecordView) string {
	if record.Selection == nil {
		return ""
	}
	return strings.TrimSpace(record.Selection.WinnerPath)
}

func summaryCountEquals(raw string, key string, want int) bool {
	got, ok := summaryCount(raw, key)
	return ok && got == want
}

func summaryCountAtLeast(raw string, key string, min int) bool {
	got, ok := summaryCount(raw, key)
	return ok && got >= min
}

func summaryCount(raw string, key string) (int, bool) {
	prefix := "- " + key + ": "
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		valueText := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		value, err := strconv.Atoi(valueText)
		if err != nil {
			return 0, false
		}
		return value, true
	}
	return 0, false
}

type sessionStateSummary struct {
	ProjectionGeneration int    `json:"projection_generation"`
	ControlPlaneEvents   int    `json:"control_plane_events"`
	LastControlPlaneKind string `json:"last_control_plane_kind"`
}

func decodeSessionStateSummary(raw []byte) (sessionStateSummary, error) {
	var state sessionStateSummary
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, err
	}
	return state, nil
}

func summaryLineValue(raw string, key string) (string, bool) {
	prefix := "- " + key + ": "
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
	}
	return "", false
}

func projectionHasMaterializationState(record projectionRecordView) bool {
	if record.Materialization == nil {
		return false
	}
	return strings.TrimSpace(record.Materialization.State) != ""
}

func projectionMaterializationStateIn(record projectionRecordView, states ...string) bool {
	if record.Materialization == nil {
		return false
	}
	state := strings.TrimSpace(record.Materialization.State)
	if state == "" {
		return false
	}
	for _, candidate := range states {
		if state == candidate {
			return true
		}
	}
	return false
}

func projectionHasMaterializationFailureDetail(record projectionRecordView) bool {
	if record.Materialization == nil {
		return false
	}
	if strings.TrimSpace(record.Materialization.Reason) != "" {
		return true
	}
	if record.Materialization.Failure == nil {
		return false
	}
	return strings.TrimSpace(record.Materialization.Failure.Code) != ""
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

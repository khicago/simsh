package main

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
	"time"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
	"github.com/khicago/simsh/pkg/contract"
)

func TestTaskManifestMatchesSupportedScenarioScope(t *testing.T) {
	t.Parallel()

	manifest, err := LoadTaskManifest(DefaultTaskManifestPath)
	if err != nil {
		t.Fatalf("LoadTaskManifest(%q) failed: %v", DefaultTaskManifestPath, err)
	}
	inventory, err := externalmapping.LoadScenarioInventory("")
	if err != nil {
		t.Fatalf("LoadScenarioInventory(...) failed: %v", err)
	}
	if err := validateTaskManifest(manifest, inventory); err != nil {
		t.Fatalf("validateTaskManifest(...) failed: %v", err)
	}

	got := make([]string, 0, len(manifest.Tasks))
	for _, task := range manifest.Tasks {
		got = append(got, task.ScenarioID)
		if _, ok := inventory.LookupScenario(task.ScenarioID); !ok {
			t.Fatalf("task manifest scenario %q is missing from inventory", task.ScenarioID)
		}
	}
	want := supportedScenarioIDs()
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("task manifest scenarios = %v, want %v", got, want)
	}
}

func TestRunPairedUpliftProducesExpectedOutcomes(t *testing.T) {
	manifest, err := LoadTaskManifest(DefaultTaskManifestPath)
	if err != nil {
		t.Fatalf("LoadTaskManifest(%q) failed: %v", DefaultTaskManifestPath, err)
	}

	snapshot, err := RunPairedUplift(context.Background(), manifest, DefaultTaskManifestPath)
	if err != nil {
		t.Fatalf("RunPairedUplift(...) failed: %v", err)
	}
	if len(snapshot.Tasks) != 3 {
		t.Fatalf("RunPairedUplift(...).Tasks = %d, want 3", len(snapshot.Tasks))
	}

	var relative, inspect, planning *PairRunRecord
	for idx := range snapshot.Tasks {
		task := &snapshot.Tasks[idx]
		switch task.ScenarioID {
		case "relative_navigation_session":
			relative = task
		case "inspect_edit_write_loop":
			inspect = task
		case "trace_consumable_planning":
			planning = task
		}
	}
	if relative == nil || inspect == nil || planning == nil {
		t.Fatalf("RunPairedUplift(...) did not return the expected scenario set: %+v", snapshot.Tasks)
	}
	if !relative.Simsh.Success || !relative.Baseline.Success {
		t.Fatalf("relative_navigation_session success mismatch: simsh=%t baseline=%t", relative.Simsh.Success, relative.Baseline.Success)
	}
	if relative.Baseline.LastMisunderstandingKind != misunderstandingNoSessionCWD {
		t.Fatalf("relative_navigation_session baseline last misunderstanding = %q, want %q", relative.Baseline.LastMisunderstandingKind, misunderstandingNoSessionCWD)
	}
	if !inspect.Simsh.Success || !inspect.Baseline.Success {
		t.Fatalf("inspect_edit_write_loop success mismatch: simsh=%t baseline=%t", inspect.Simsh.Success, inspect.Baseline.Success)
	}
	if inspect.Baseline.LastMisunderstandingKind != misunderstandingMissingRG {
		t.Fatalf("inspect_edit_write_loop baseline last misunderstanding = %q, want %q", inspect.Baseline.LastMisunderstandingKind, misunderstandingMissingRG)
	}
	if !planning.Simsh.Success {
		t.Fatalf("trace_consumable_planning simsh success = false, want true")
	}
	if planning.Baseline.Success {
		t.Fatalf("trace_consumable_planning baseline success = true, want false")
	}
	if planning.Baseline.FailureKind != failureKindBudgetAfterFallback {
		t.Fatalf("trace_consumable_planning baseline failure kind = %q, want %q", planning.Baseline.FailureKind, failureKindBudgetAfterFallback)
	}
	if planning.Baseline.LastMisunderstandingKind != misunderstandingMissingJSON {
		t.Fatalf("trace_consumable_planning baseline last misunderstanding = %q, want %q", planning.Baseline.LastMisunderstandingKind, misunderstandingMissingJSON)
	}
}

func TestClassifyCommandSurfaceUnavailableUsesStructuredExternalOutcome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		result         contract.ExecutionResult
		command        string
		wantMissing    bool
		wantStructured bool
		wantKind       contract.ExternalOutcomeKind
	}{
		{
			name: "command not found",
			result: contract.ExecutionResult{
				ExitCode: contract.ExitCodeGeneral,
				Stdout:   "json: command failed for a different reason",
				Trace: contract.ExecutionTrace{
					ExternalOutcomes: []contract.ExecutionTraceStep{
						{Command: "json", OutcomeKind: contract.ExternalOutcomeCommandNotFound},
					},
				},
			},
			command:        "json",
			wantMissing:    true,
			wantStructured: true,
			wantKind:       contract.ExternalOutcomeCommandNotFound,
		},
		{
			name: "unsupported command",
			result: contract.ExecutionResult{
				ExitCode: contract.ExitCodeUnsupported,
				Trace: contract.ExecutionTrace{
					ExternalOutcomes: []contract.ExecutionTraceStep{
						{ResolvedPath: contract.VirtualExternalBinDir + "/" + "rg", OutcomeKind: contract.ExternalOutcomeUnsupported},
					},
				},
			},
			command:        "rg",
			wantMissing:    true,
			wantStructured: true,
			wantKind:       contract.ExternalOutcomeUnsupported,
		},
		{
			name: "structured success defeats compatibility text",
			result: contract.ExecutionResult{
				ExitCode: contract.ExitCodeGeneral,
				Stdout:   "json: not found",
				Trace: contract.ExecutionTrace{
					ExternalOutcomes: []contract.ExecutionTraceStep{
						{Argv: []string{"json"}, OutcomeKind: contract.ExternalOutcomeSuccess},
					},
				},
			},
			command:        "json",
			wantMissing:    false,
			wantStructured: true,
			wantKind:       contract.ExternalOutcomeSuccess,
		},
		{
			name: "legacy compatibility fallback without structured outcome",
			result: contract.ExecutionResult{
				ExitCode: contract.ExitCodeGeneral,
				Stderr:   "rg: not supported",
			},
			command:     "rg",
			wantMissing: true,
		},
		{
			name: "wrong command outcome does not classify",
			result: contract.ExecutionResult{
				ExitCode: contract.ExitCodeGeneral,
				Trace: contract.ExecutionTrace{
					ExternalOutcomes: []contract.ExecutionTraceStep{
						{Command: "other", OutcomeKind: contract.ExternalOutcomeCommandNotFound},
					},
				},
			},
			command: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := classifyCommandSurfaceUnavailable(tt.result, tt.command)
			if got.Missing != tt.wantMissing {
				t.Fatalf("classifyCommandSurfaceUnavailable(%q).Missing = %t, want %t; signal=%+v", tt.command, got.Missing, tt.wantMissing, got)
			}
			if got.UsedStructured != tt.wantStructured {
				t.Fatalf("classifyCommandSurfaceUnavailable(%q).UsedStructured = %t, want %t; signal=%+v", tt.command, got.UsedStructured, tt.wantStructured, got)
			}
			if got.OutcomeKind != tt.wantKind {
				t.Fatalf("classifyCommandSurfaceUnavailable(%q).OutcomeKind = %q, want %q; signal=%+v", tt.command, got.OutcomeKind, tt.wantKind, got)
			}
		})
	}
}

func TestRecordStepKeepsExternalOutcomeBreadcrumb(t *testing.T) {
	t.Parallel()

	state := newTaskExecutionState(substrateThinCoreStateless, PairedTaskBudget{
		MaxSteps:             3,
		MaxObservationTokens: 100,
	})
	exitCode := contract.ExitCodeUnsupported
	result := contract.ExecutionResult{
		ExitCode: exitCode,
		Trace: contract.ExecutionTrace{
			ExternalOutcomes: []contract.ExecutionTraceStep{
				{
					Command:      "json",
					ResolvedPath: contract.VirtualExternalBinDir + "/" + "json",
					OutcomeKind:  contract.ExternalOutcomeUnsupported,
					ExitCode:     &exitCode,
				},
			},
		},
	}

	state.recordStep("read_task_count", "json len --fmt json", result, classificationMisunderstandingWithSource(misunderstandingMissingJSON, "json unavailable", classificationSourceStructured))

	if len(state.run.StepsDetail) != 1 {
		t.Fatalf("recordStep(...).StepsDetail length = %d, want 1", len(state.run.StepsDetail))
	}
	step := state.run.StepsDetail[0]
	if step.ClassificationSource != classificationSourceStructured {
		t.Fatalf("recordStep(...).ClassificationSource = %q, want %q", step.ClassificationSource, classificationSourceStructured)
	}
	if len(step.ExternalOutcomes) != 1 {
		t.Fatalf("recordStep(...).ExternalOutcomes length = %d, want 1; step=%+v", len(step.ExternalOutcomes), step)
	}
	outcome := step.ExternalOutcomes[0]
	if outcome.Command != "json" || outcome.ResolvedPath != contract.VirtualExternalBinDir+"/"+"json" || outcome.OutcomeKind != string(contract.ExternalOutcomeUnsupported) {
		t.Fatalf("recordStep(...).ExternalOutcomes[0] = %+v, want json unsupported breadcrumb", outcome)
	}
	if outcome.ExitCode == nil || *outcome.ExitCode != contract.ExitCodeUnsupported {
		t.Fatalf("recordStep(...).ExternalOutcomes[0].ExitCode = %v, want %d", outcome.ExitCode, contract.ExitCodeUnsupported)
	}
}

func TestRecordStepSanitizesExternalOutcomeBreadcrumbPaths(t *testing.T) {
	t.Parallel()

	state := newTaskExecutionState(substrateThinCoreStateless, PairedTaskBudget{
		MaxSteps:             3,
		MaxObservationTokens: 100,
	})
	result := contract.ExecutionResult{
		ExitCode: contract.ExitCodeGeneral,
		Trace: contract.ExecutionTrace{
			ExternalOutcomes: []contract.ExecutionTraceStep{
				{
					Command:      "host_tool",
					ResolvedPath: "/tmp/provider/bin/host_tool",
					OutcomeKind:  contract.ExternalOutcomeProviderFailure,
				},
				{
					Command:      "rg",
					ResolvedPath: path.Join(contract.VirtualExternalBinDir, "rg"),
					OutcomeKind:  contract.ExternalOutcomeUnsupported,
				},
				{
					Command:      "win_tool",
					ResolvedPath: `C:\tools\win_tool.exe`,
					OutcomeKind:  contract.ExternalOutcomeProviderFailure,
				},
				{
					Command:      "json",
					ResolvedPath: "json",
					OutcomeKind:  contract.ExternalOutcomeCommandNotFound,
				},
			},
		},
	}

	state.recordStep("inspect", "host_tool && rg && json", result, classificationProgress("checked external outcomes"))

	if len(state.run.StepsDetail) != 1 {
		t.Fatalf("recordStep(...).StepsDetail length = %d, want 1", len(state.run.StepsDetail))
	}
	outcomes := state.run.StepsDetail[0].ExternalOutcomes
	if len(outcomes) != 4 {
		t.Fatalf("recordStep(...).ExternalOutcomes length = %d, want 4; outcomes=%+v", len(outcomes), outcomes)
	}
	if outcomes[0].Command != "host_tool" || outcomes[0].ResolvedPath != "" {
		t.Fatalf("host-local external outcome breadcrumb = %+v, want command preserved with empty resolved path", outcomes[0])
	}
	wantRGPath := path.Join(contract.VirtualExternalBinDir, "rg")
	if outcomes[1].ResolvedPath != wantRGPath {
		t.Fatalf("virtual external resolved path = %q, want %q", outcomes[1].ResolvedPath, wantRGPath)
	}
	if outcomes[2].Command != "win_tool" || outcomes[2].ResolvedPath != "" {
		t.Fatalf("windows-local external outcome breadcrumb = %+v, want command preserved with empty resolved path", outcomes[2])
	}
	if outcomes[3].ResolvedPath != "json" {
		t.Fatalf("bare command resolved path = %q, want json", outcomes[3].ResolvedPath)
	}
}

func TestBuildFailureTaxonomyRollup(t *testing.T) {
	t.Parallel()

	snapshot := PairedRunSnapshot{
		GeneratedAt:       mustStaticTime("2026-04-06T00:00:00Z"),
		ComparisonRule:    pairedUpliftComparisonRule,
		TaskManifestPath:  DefaultTaskManifestPath,
		AgentID:           pairedProbeAgentID,
		BaselineSubstrate: substrateThinCoreStateless,
		SimshSubstrate:    substrateSimshFullSessioned,
		Tasks: []PairRunRecord{
			{
				ScenarioID: "inspect_edit_write_loop",
				Simsh: SubstrateRunRecord{
					Substrate: substrateSimshFullSessioned,
				},
				Baseline: SubstrateRunRecord{
					Substrate:   substrateThinCoreStateless,
					FailureKind: failureKindBudgetAfterFallback,
					StepsDetail: []StepRecord{
						{
							EnvironmentMisunderstood: true,
							MisunderstandingKind:     misunderstandingMissingRG,
							ClassificationSource:     classificationSourceStructured,
							ExternalOutcomes:         []ExternalOutcomeSummary{{OutcomeKind: string(contract.ExternalOutcomeUnsupported)}},
						},
						{
							EnvironmentMisunderstood: true,
							MisunderstandingKind:     misunderstandingMissingRG,
							ClassificationSource:     classificationSourceStructured,
							ExternalOutcomes:         []ExternalOutcomeSummary{{OutcomeKind: string(contract.ExternalOutcomeUnsupported)}},
						},
					},
				},
			},
			{
				ScenarioID: "trace_consumable_planning",
				Baseline: SubstrateRunRecord{
					Substrate: substrateThinCoreStateless,
					StepsDetail: []StepRecord{
						{
							EnvironmentMisunderstood: true,
							MisunderstandingKind:     misunderstandingMissingJSON,
							ClassificationSource:     classificationSourceCompatText,
						},
					},
				},
			},
		},
	}

	report := BuildFailureTaxonomy(snapshot, DefaultSnapshotPath)
	if len(report.Entries) != 3 {
		t.Fatalf("BuildFailureTaxonomy(...).Entries = %d, want 3", len(report.Entries))
	}
	lookup := make(map[string]FailureTaxonomyEntry, len(report.Entries))
	for _, entry := range report.Entries {
		lookup[entry.Bucket+"|"+entry.Runtime+"|"+entry.Kind] = entry
	}
	failureEntry, ok := lookup[taxonomyBucketFailure+"|"+substrateThinCoreStateless+"|"+failureKindBudgetAfterFallback]
	if !ok {
		t.Fatalf("failure taxonomy missing budget exhaustion entry: %+v", report.Entries)
	}
	if failureEntry.Count != 1 {
		t.Fatalf("budget exhaustion failure count = %d, want 1", failureEntry.Count)
	}
	jsonEntry, ok := lookup[taxonomyBucketMisunderstanding+"|"+substrateThinCoreStateless+"|"+misunderstandingMissingJSON]
	if !ok {
		t.Fatalf("failure taxonomy missing json misunderstanding entry: %+v", report.Entries)
	}
	if jsonEntry.Count != 1 {
		t.Fatalf("json misunderstanding count = %d, want 1", jsonEntry.Count)
	}
	rgEntry, ok := lookup[taxonomyBucketMisunderstanding+"|"+substrateThinCoreStateless+"|"+misunderstandingMissingRG]
	if !ok {
		t.Fatalf("failure taxonomy missing rg misunderstanding entry: %+v", report.Entries)
	}
	if rgEntry.Count != 2 {
		t.Fatalf("rg misunderstanding count = %d, want 2", rgEntry.Count)
	}
	if !slices.Equal(rgEntry.ClassificationSources, []string{classificationSourceStructured}) {
		t.Fatalf("rg classification sources = %v, want structured source", rgEntry.ClassificationSources)
	}
	if !slices.Equal(rgEntry.ExternalOutcomeKinds, []string{string(contract.ExternalOutcomeUnsupported)}) {
		t.Fatalf("rg external outcome kinds = %v, want unsupported", rgEntry.ExternalOutcomeKinds)
	}
	if !slices.Equal(jsonEntry.ClassificationSources, []string{classificationSourceCompatText}) {
		t.Fatalf("json classification sources = %v, want compatibility text source", jsonEntry.ClassificationSources)
	}
}

func TestCheckedInRawSnapshotMatchesTaskManifest(t *testing.T) {
	t.Parallel()

	manifest, err := LoadTaskManifest(DefaultTaskManifestPath)
	if err != nil {
		t.Fatalf("LoadTaskManifest(%q) failed: %v", DefaultTaskManifestPath, err)
	}
	snapshot, err := LoadPairedRunSnapshot(DefaultSnapshotPath)
	if err != nil {
		t.Fatalf("LoadPairedRunSnapshot(%q) failed: %v", DefaultSnapshotPath, err)
	}

	if snapshot.TaskManifestPath != DefaultTaskManifestPath {
		t.Errorf("LoadPairedRunSnapshot(%q).TaskManifestPath = %q, want %q", DefaultSnapshotPath, snapshot.TaskManifestPath, DefaultTaskManifestPath)
	}
	if snapshot.ComparisonRule != manifest.ComparisonRule {
		t.Errorf("LoadPairedRunSnapshot(%q).ComparisonRule = %q, want %q", DefaultSnapshotPath, snapshot.ComparisonRule, manifest.ComparisonRule)
	}
	if snapshot.AgentID != manifest.AgentID {
		t.Errorf("LoadPairedRunSnapshot(%q).AgentID = %q, want %q", DefaultSnapshotPath, snapshot.AgentID, manifest.AgentID)
	}
	if snapshot.BaselineSubstrate != manifest.BaselineSubstrate {
		t.Errorf("LoadPairedRunSnapshot(%q).BaselineSubstrate = %q, want %q", DefaultSnapshotPath, snapshot.BaselineSubstrate, manifest.BaselineSubstrate)
	}
	if snapshot.SimshSubstrate != manifest.SimshSubstrate {
		t.Errorf("LoadPairedRunSnapshot(%q).SimshSubstrate = %q, want %q", DefaultSnapshotPath, snapshot.SimshSubstrate, manifest.SimshSubstrate)
	}
	if len(snapshot.Tasks) != len(manifest.Tasks) {
		t.Fatalf("LoadPairedRunSnapshot(%q).Tasks length = %d, want %d", DefaultSnapshotPath, len(snapshot.Tasks), len(manifest.Tasks))
	}

	manifestByScenario := make(map[string]PairedTaskManifest, len(manifest.Tasks))
	for _, task := range manifest.Tasks {
		manifestByScenario[task.ScenarioID] = task
	}
	for _, task := range snapshot.Tasks {
		want, ok := manifestByScenario[task.ScenarioID]
		if !ok {
			t.Errorf("LoadPairedRunSnapshot(%q) includes scenario %q missing from %q", DefaultSnapshotPath, task.ScenarioID, DefaultTaskManifestPath)
			continue
		}
		delete(manifestByScenario, task.ScenarioID)
		if task.AgentID != manifest.AgentID {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].AgentID = %q, want %q", DefaultSnapshotPath, task.ScenarioID, task.AgentID, manifest.AgentID)
		}
		if task.PairSeed != want.PairSeed {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].PairSeed = %d, want %d", DefaultSnapshotPath, task.ScenarioID, task.PairSeed, want.PairSeed)
		}
		if task.RunOrder != want.RunOrder {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].RunOrder = %q, want %q", DefaultSnapshotPath, task.ScenarioID, task.RunOrder, want.RunOrder)
		}
		if task.Budget.MaxSteps != want.MaxSteps {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].Budget.MaxSteps = %d, want %d", DefaultSnapshotPath, task.ScenarioID, task.Budget.MaxSteps, want.MaxSteps)
		}
		if task.Budget.MaxObservationTokens != want.MaxObservationTokens {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].Budget.MaxObservationTokens = %d, want %d", DefaultSnapshotPath, task.ScenarioID, task.Budget.MaxObservationTokens, want.MaxObservationTokens)
		}
		if !slices.Equal(task.ExpectedOutputs, want.ExpectedOutputs) {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].ExpectedOutputs = %v, want %v", DefaultSnapshotPath, task.ScenarioID, task.ExpectedOutputs, want.ExpectedOutputs)
		}
		if task.WhySelected != want.WhySelected {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].WhySelected = %q, want %q", DefaultSnapshotPath, task.ScenarioID, task.WhySelected, want.WhySelected)
		}
		wantRefs := []string{
			DefaultTaskManifestPath + "#" + task.ScenarioID,
			externalmapping.DefaultScenarioInventoryPath + "#" + task.ScenarioID,
		}
		if !slices.Equal(task.EvidenceRefs, wantRefs) {
			t.Errorf("LoadPairedRunSnapshot(%q).Tasks[%q].EvidenceRefs = %v, want %v", DefaultSnapshotPath, task.ScenarioID, task.EvidenceRefs, wantRefs)
		}
	}
	if len(manifestByScenario) != 0 {
		missing := make([]string, 0, len(manifestByScenario))
		for scenarioID := range manifestByScenario {
			missing = append(missing, scenarioID)
		}
		slices.Sort(missing)
		t.Errorf("LoadPairedRunSnapshot(%q) is missing manifest scenarios: %v", DefaultSnapshotPath, missing)
	}
}

func TestCheckedInReportsMatchSnapshot(t *testing.T) {
	t.Parallel()

	snapshot, err := LoadPairedRunSnapshot(DefaultSnapshotPath)
	if err != nil {
		t.Fatalf("LoadPairedRunSnapshot(%q) failed: %v", DefaultSnapshotPath, err)
	}
	artifact, err := BuildPairedUpliftArtifact(snapshot)
	if err != nil {
		t.Fatalf("BuildPairedUpliftArtifact(...) failed: %v", err)
	}
	taxonomy := BuildFailureTaxonomy(snapshot, DefaultSnapshotPath)

	gotArtifact, err := MarshalPairedUpliftArtifactJSON(artifact)
	if err != nil {
		t.Fatalf("MarshalPairedUpliftArtifactJSON(...) failed: %v", err)
	}
	gotSummary := []byte(RenderPairedUpliftMarkdown(artifact, taxonomy))
	gotFailures, err := MarshalFailureTaxonomyJSON(taxonomy)
	if err != nil {
		t.Fatalf("MarshalFailureTaxonomyJSON(...) failed: %v", err)
	}

	wantArtifact, err := readReportFile(DefaultArtifactPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", DefaultArtifactPath, err)
	}
	wantSummary, err := readReportFile(DefaultSummaryPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", DefaultSummaryPath, err)
	}
	wantFailures, err := readReportFile(DefaultFailureTaxonomyPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", DefaultFailureTaxonomyPath, err)
	}

	if string(gotArtifact) != string(wantArtifact) {
		t.Fatalf("aggregate artifact drifted from checked-in report")
	}
	if string(gotSummary) != string(wantSummary) {
		t.Fatalf("markdown summary drifted from checked-in report")
	}
	if string(gotFailures) != string(wantFailures) {
		t.Fatalf("failure taxonomy drifted from checked-in report")
	}
}

func mustStaticTime(raw string) time.Time {
	tm, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic(err)
	}
	return tm
}

func readReportFile(path string) ([]byte, error) {
	candidates := []string{
		path,
		filepath.Base(path),
		filepath.Join("reports", filepath.Base(path)),
		filepath.Join("benchmarks", "paired_uplift", "reports", filepath.Base(path)),
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}

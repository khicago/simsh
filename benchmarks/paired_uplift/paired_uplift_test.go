package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
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
						{EnvironmentMisunderstood: true, MisunderstandingKind: misunderstandingMissingRG},
						{EnvironmentMisunderstood: true, MisunderstandingKind: misunderstandingMissingRG},
					},
				},
			},
			{
				ScenarioID: "trace_consumable_planning",
				Baseline: SubstrateRunRecord{
					Substrate: substrateThinCoreStateless,
					StepsDetail: []StepRecord{
						{EnvironmentMisunderstood: true, MisunderstandingKind: misunderstandingMissingJSON},
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

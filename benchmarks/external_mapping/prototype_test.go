package externalmapping

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTerminalBenchComparison(t *testing.T) {
	t.Parallel()

	scope, err := LoadTerminalBenchPrototypeScope("")
	if err != nil {
		t.Fatalf("LoadTerminalBenchPrototypeScope() error = %v", err)
	}
	inventory, err := LoadScenarioInventory("")
	if err != nil {
		t.Fatalf("LoadScenarioInventory() error = %v", err)
	}
	mapping, err := LoadFamilyMapping(DefaultTerminalBenchMappingPath)
	if err != nil {
		t.Fatalf("LoadFamilyMapping() error = %v", err)
	}

	traceComplete := 1.0
	source := NativeSuiteReport{
		Scenarios: []NativeScenarioReport{
			{
				Name:                  "inspect_edit_write_loop",
				Category:              "file_inspect_edit_write_loops",
				Success:               true,
				SessionScoped:         true,
				AsyncCandidate:        true,
				PatchWorkflow:         true,
				DurationMS:            25,
				TraceCompleteness:     &traceComplete,
				AssertionCompleteness: &traceComplete,
			},
			{
				Name:                  "relative_navigation_session",
				Category:              "relative_path_navigation",
				Success:               true,
				SessionScoped:         true,
				AsyncCandidate:        false,
				PatchWorkflow:         false,
				DurationMS:            15,
				TraceCompleteness:     &traceComplete,
				AssertionCompleteness: &traceComplete,
			},
		},
	}

	report, err := BuildTerminalBenchComparison(inventory, mapping, source, scope, "baseline.json")
	if err != nil {
		t.Fatalf("BuildTerminalBenchComparison(...) error = %v", err)
	}
	if report.ExternalFamily != terminalBenchExternalFamily {
		t.Fatalf("report.ExternalFamily = %q, want %q", report.ExternalFamily, terminalBenchExternalFamily)
	}
	if report.Summary.ComparedScenarios != 2 || report.Summary.DirectFitScenarios != 1 || report.Summary.TranslatedProofScenarios != 1 {
		t.Fatalf("report.Summary = %+v, want 2 compared with 1 direct + 1 translated", report.Summary)
	}
	if report.Scenarios[0].ScenarioID != "inspect_edit_write_loop" || report.Scenarios[0].Role != ComparisonRoleDirectFit {
		t.Fatalf("first scenario = %+v, want inspect_edit_write_loop/direct_fit", report.Scenarios[0])
	}
	if report.Scenarios[1].ScenarioID != "relative_navigation_session" || report.Scenarios[1].Role != ComparisonRoleTranslatedProof {
		t.Fatalf("second scenario = %+v, want relative_navigation_session/translated_proof", report.Scenarios[1])
	}
}

func TestBuildTerminalBenchComparisonRejectsScopeDrift(t *testing.T) {
	t.Parallel()

	scope := TerminalBenchPrototypeScope{
		ExternalFamily: terminalBenchExternalFamily,
		ComparisonRule: "one_direct_fit_plus_one_translated_slice",
		Scenarios: []TerminalBenchPrototypeItem{
			{
				ScenarioID:           "inspect_edit_write_loop",
				Role:                 ComparisonRoleDirectFit,
				ExpectedStatus:       MappingStatusTranslated,
				ExternalTask:         "terminal_file_inspect_edit_loop",
				ComparisonGoal:       PrototypeComparisonGoalDirectFit,
				ComparableDimensions: []string{"edit_loop"},
			},
			{
				ScenarioID:           "relative_navigation_session",
				Role:                 ComparisonRoleTranslatedProof,
				ExpectedStatus:       MappingStatusTranslated,
				ExternalTask:         "terminal_relative_navigation_subflow",
				ComparisonGoal:       PrototypeComparisonGoalTranslate,
				ComparableDimensions: []string{"navigation"},
			},
		},
	}
	inventory := ScenarioInventory{
		SourceBenchmark: "simsh_native_reference",
		Scenarios: []ScenarioInventoryRecord{
			{ID: "inspect_edit_write_loop", Category: "file_inspect_edit_write_loops"},
			{ID: "relative_navigation_session", Category: "relative_path_navigation"},
		},
	}
	mapping := FamilyMapping{
		ExternalFamily: terminalBenchExternalFamily,
		Scenarios: []FamilyMappingScenario{
			{ScenarioID: "inspect_edit_write_loop", Status: MappingStatusAsIs},
			{ScenarioID: "relative_navigation_session", Status: MappingStatusTranslated},
		},
	}
	source := NativeSuiteReport{
		Scenarios: []NativeScenarioReport{
			{Name: "inspect_edit_write_loop", Category: "file_inspect_edit_write_loops"},
			{Name: "relative_navigation_session", Category: "relative_path_navigation"},
		},
	}

	if _, err := BuildTerminalBenchComparison(inventory, mapping, source, scope, DefaultNativeBaselineReportPath); err == nil {
		t.Fatal("BuildTerminalBenchComparison(...) error = nil, want scope drift error")
	}
}

func TestCheckedInTerminalBenchComparisonMatchesGenerator(t *testing.T) {
	scope, err := LoadTerminalBenchPrototypeScope("")
	if err != nil {
		t.Fatalf("LoadTerminalBenchPrototypeScope() error = %v", err)
	}
	inventory, err := LoadScenarioInventory("")
	if err != nil {
		t.Fatalf("LoadScenarioInventory() error = %v", err)
	}
	mapping, err := LoadFamilyMapping(DefaultTerminalBenchMappingPath)
	if err != nil {
		t.Fatalf("LoadFamilyMapping() error = %v", err)
	}
	source, err := LoadNativeSuiteReport(DefaultNativeBaselineReportPath)
	if err != nil {
		t.Fatalf("LoadNativeSuiteReport() error = %v", err)
	}
	report, err := BuildTerminalBenchComparison(inventory, mapping, source, scope, DefaultNativeBaselineReportPath)
	if err != nil {
		t.Fatalf("BuildTerminalBenchComparison(...) error = %v", err)
	}
	want, err := MarshalTerminalBenchComparisonJSON(report)
	if err != nil {
		t.Fatalf("MarshalTerminalBenchComparisonJSON(...) error = %v", err)
	}
	candidates := []string{
		filepath.Clean(DefaultTerminalBenchArtifactPath),
		filepath.Join("..", "terminal_bench_compare", "reports", filepath.Base(DefaultTerminalBenchArtifactPath)),
	}
	var got []byte
	for _, candidate := range candidates {
		got, err = os.ReadFile(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("read checked-in comparison artifact: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("checked-in comparison artifact drifted from generator")
	}
}

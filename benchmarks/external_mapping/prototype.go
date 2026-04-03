package externalmapping

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func BuildTerminalBenchComparison(
	inventory ScenarioInventory,
	mapping FamilyMapping,
	report NativeSuiteReport,
	scope TerminalBenchPrototypeScope,
	reportPath string,
) (TerminalBenchComparisonArtifact, error) {
	if mapping.ExternalFamily != terminalBenchExternalFamily {
		return TerminalBenchComparisonArtifact{}, fmt.Errorf("expected %s mapping, got %q", terminalBenchExternalFamily, mapping.ExternalFamily)
	}
	if scope.ExternalFamily != terminalBenchExternalFamily {
		return TerminalBenchComparisonArtifact{}, fmt.Errorf("expected %s scope, got %q", terminalBenchExternalFamily, scope.ExternalFamily)
	}
	if scope.ComparisonRule != "one_direct_fit_plus_one_translated_slice" {
		return TerminalBenchComparisonArtifact{}, fmt.Errorf("unexpected comparison rule %q", scope.ComparisonRule)
	}
	if len(scope.Scenarios) != 2 {
		return TerminalBenchComparisonArtifact{}, fmt.Errorf("prototype scope must contain exactly 2 scenarios, got %d", len(scope.Scenarios))
	}

	inventoryByID := make(map[string]ScenarioInventoryRecord, len(inventory.Scenarios))
	for _, scenario := range inventory.Scenarios {
		inventoryByID[scenario.ID] = scenario
	}
	mappingByID := make(map[string]FamilyMappingScenario, len(mapping.Scenarios))
	for _, scenario := range mapping.Scenarios {
		mappingByID[scenario.ScenarioID] = scenario
	}
	reportByID := make(map[string]NativeScenarioReport, len(report.Scenarios))
	for _, scenario := range report.Scenarios {
		reportByID[scenario.Name] = scenario
	}

	compared := make([]TerminalBenchComparedScenario, 0, len(scope.Scenarios))
	directCount := 0
	translatedCount := 0
	for _, scoped := range scope.Scenarios {
		switch scoped.Role {
		case ComparisonRoleDirectFit:
			directCount++
		case ComparisonRoleTranslatedProof:
			translatedCount++
		default:
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("unexpected scope role %q", scoped.Role)
		}

		inventoryScenario, ok := inventoryByID[scoped.ScenarioID]
		if !ok {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q missing from inventory", scoped.ScenarioID)
		}
		mappingScenario, ok := mappingByID[scoped.ScenarioID]
		if !ok {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q missing from terminal mapping", scoped.ScenarioID)
		}
		if mappingScenario.Status != scoped.ExpectedStatus {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q expected mapping status %q, got %q", scoped.ScenarioID, scoped.ExpectedStatus, mappingScenario.Status)
		}
		if scoped.ExpectedStatus != MappingStatusAsIs && scoped.ExpectedStatus != MappingStatusTranslated {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q expected status %q, want as_is|translated", scoped.ScenarioID, scoped.ExpectedStatus)
		}
		if strings.TrimSpace(scoped.ExternalTask) == "" {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q missing external_task", scoped.ScenarioID)
		}
		if len(scoped.ComparableDimensions) == 0 {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q missing comparable_dimensions", scoped.ScenarioID)
		}
		reportScenario, ok := reportByID[scoped.ScenarioID]
		if !ok {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q missing from native suite report", scoped.ScenarioID)
		}
		if reportScenario.Category != inventoryScenario.Category {
			return TerminalBenchComparisonArtifact{}, fmt.Errorf("scenario %q native report category = %q, want %q", scoped.ScenarioID, reportScenario.Category, inventoryScenario.Category)
		}

		compared = append(compared, TerminalBenchComparedScenario{
			ScenarioID:           scoped.ScenarioID,
			Category:             inventoryScenario.Category,
			TaskShape:            inventoryScenario.TaskShape,
			Role:                 scoped.Role,
			MappingStatus:        mappingScenario.Status,
			ComparisonGoal:       scoped.ComparisonGoal,
			ExternalTask:         scoped.ExternalTask,
			Summary:              inventoryScenario.Summary,
			TruthSurfaces:        append([]string(nil), inventoryScenario.TruthSurfaces...),
			ComparableDimensions: append([]string(nil), scoped.ComparableDimensions...),
			ExcludedDimensions:   append([]string(nil), scoped.ExcludedDimensions...),
			Rationale:            mappingScenario.Rationale,
			TranslationNotes:     strings.TrimSpace(mappingScenario.TranslationNotes),
			WhySelected:          strings.TrimSpace(scoped.WhySelected),
			EvidenceRefs: []string{
				DefaultScenarioInventoryPath + "#" + scoped.ScenarioID,
				DefaultTerminalBenchMappingPath + "#" + scoped.ScenarioID,
				DefaultTerminalBenchPrototypeScopePath + "#" + scoped.ScenarioID,
				reportPath + "#" + scoped.ScenarioID,
			},
			NativeResult: TerminalBenchNativeResult{
				Success:               reportScenario.Success,
				SessionScoped:         reportScenario.SessionScoped,
				AsyncCandidate:        reportScenario.AsyncCandidate,
				PatchWorkflow:         reportScenario.PatchWorkflow,
				DurationMS:            reportScenario.DurationMS,
				TraceCompleteness:     reportScenario.TraceCompleteness,
				AssertionCompleteness: reportScenario.AssertionCompleteness,
				Notes:                 append([]string(nil), reportScenario.Notes...),
			},
		})
	}

	if directCount != 1 || translatedCount != 1 {
		return TerminalBenchComparisonArtifact{}, fmt.Errorf("prototype scope must contain exactly 1 direct_fit and 1 translated_proof slice, got direct=%d translated=%d", directCount, translatedCount)
	}

	slices.SortFunc(compared, func(a, b TerminalBenchComparedScenario) int {
		return strings.Compare(a.Role+a.ScenarioID, b.Role+b.ScenarioID)
	})

	return TerminalBenchComparisonArtifact{
		Version:        1,
		ExternalFamily: terminalBenchExternalFamily,
		Source: TerminalBenchComparisonSource{
			Benchmark:   inventory.SourceBenchmark,
			ReportPath:  reportPath,
			GeneratedAt: report.GeneratedAt,
		},
		Scope: scope,
		Summary: TerminalBenchComparisonSummary{
			ComparedScenarios:        len(compared),
			DirectFitScenarios:       directCount,
			TranslatedProofScenarios: translatedCount,
			AllNativeSuccessful:      comparedScenariosSuccessful(compared),
		},
		Scenarios: compared,
	}, nil
}

func BuildDefaultTerminalBenchComparison(reportPath string) (TerminalBenchComparisonArtifact, error) {
	inventory, err := LoadScenarioInventory("")
	if err != nil {
		return TerminalBenchComparisonArtifact{}, err
	}
	mapping, err := LoadFamilyMapping(DefaultTerminalBenchMappingPath)
	if err != nil {
		return TerminalBenchComparisonArtifact{}, err
	}
	report, err := LoadNativeSuiteReport(reportPath)
	if err != nil {
		return TerminalBenchComparisonArtifact{}, err
	}
	scope, err := LoadTerminalBenchPrototypeScope("")
	if err != nil {
		return TerminalBenchComparisonArtifact{}, err
	}
	if reportPath == "" {
		reportPath = DefaultNativeBaselineReportPath
	}
	return BuildTerminalBenchComparison(inventory, mapping, report, scope, reportPath)
}

func comparedScenariosSuccessful(compared []TerminalBenchComparedScenario) bool {
	for _, scenario := range compared {
		if !scenario.NativeResult.Success {
			return false
		}
	}
	return true
}

func MarshalTerminalBenchComparisonJSON(artifact TerminalBenchComparisonArtifact) ([]byte, error) {
	return json.MarshalIndent(artifact, "", "  ")
}

func RenderTerminalBenchComparisonMarkdown(artifact TerminalBenchComparisonArtifact) string {
	lines := []string{
		"# Terminal-Bench Comparison Prototype",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- External family: `%s`", artifact.ExternalFamily),
		fmt.Sprintf("- Source benchmark: `%s`", artifact.Source.Benchmark),
		fmt.Sprintf("- Source report: `%s`", artifact.Source.ReportPath),
		"",
		"## Summary",
		"",
		fmt.Sprintf("- Compared scenarios: %d", artifact.Summary.ComparedScenarios),
		fmt.Sprintf("- Direct-fit slices: %d", artifact.Summary.DirectFitScenarios),
		fmt.Sprintf("- Translated proof slices: %d", artifact.Summary.TranslatedProofScenarios),
		fmt.Sprintf("- All native scenarios successful: `%t`", artifact.Summary.AllNativeSuccessful),
		"",
		"## Compared Scenarios",
		"",
		"| scenario | role | external task | status | success | patch workflow | trace completeness |",
		"| --- | --- | --- | --- | --- | --- | --- |",
	}
	for _, scenario := range artifact.Scenarios {
		traceValue := "-"
		if scenario.NativeResult.TraceCompleteness != nil {
			traceValue = fmt.Sprintf("%.2f", *scenario.NativeResult.TraceCompleteness)
		}
		lines = append(lines, fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%t` | `%t` | `%s` |",
			scenario.ScenarioID,
			scenario.Role,
			scenario.ExternalTask,
			scenario.MappingStatus,
			scenario.NativeResult.Success,
			scenario.NativeResult.PatchWorkflow,
			traceValue,
		))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Why `%s` is here: %s", scenario.ScenarioID, scenario.WhySelected))
		lines = append(lines, fmt.Sprintf("Comparable dimensions for `%s`: `%s`", scenario.ScenarioID, strings.Join(scenario.ComparableDimensions, "`, `")))
		if len(scenario.ExcludedDimensions) > 0 {
			lines = append(lines, fmt.Sprintf("Excluded dimensions for `%s`: `%s`", scenario.ScenarioID, strings.Join(scenario.ExcludedDimensions, "`, `")))
		}
		if scenario.TranslationNotes != "" {
			lines = append(lines, fmt.Sprintf("Translation notes for `%s`: %s", scenario.ScenarioID, scenario.TranslationNotes))
		}
	}
	lines = append(lines,
		"",
		"## Notes",
		"",
		"- This artifact stays downstream from the native benchmark SSOT and the checked-in Terminal-Bench mapping layer.",
		"- It is a comparison prototype, not a benchmark adoption layer and not a second scenario catalog.",
	)
	return strings.Join(lines, "\n") + "\n"
}

package main

import (
	"testing"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
)

func TestPrototypeScopeMatchesTerminalBenchMapping(t *testing.T) {
	t.Parallel()

	scope, err := externalmapping.LoadTerminalBenchPrototypeScope("")
	if err != nil {
		t.Fatalf("LoadTerminalBenchPrototypeScope() error = %v", err)
	}
	if scope.ExternalFamily != "Terminal-Bench" {
		t.Fatalf("scope.ExternalFamily = %q, want %q", scope.ExternalFamily, "Terminal-Bench")
	}
	if len(scope.Scenarios) != 2 {
		t.Fatalf("len(scope.Scenarios) = %d, want 2", len(scope.Scenarios))
	}

	inventory, err := externalmapping.LoadScenarioInventory("")
	if err != nil {
		t.Fatalf("LoadScenarioInventory() error = %v", err)
	}
	mapping, err := externalmapping.LoadFamilyMapping(externalmapping.DefaultTerminalBenchMappingPath)
	if err != nil {
		t.Fatalf("LoadFamilyMapping(...) error = %v", err)
	}

	directCount := 0
	translatedCount := 0
	for _, scenario := range scope.Scenarios {
		if _, ok := inventory.LookupScenario(scenario.ScenarioID); !ok {
			t.Fatalf("scope scenario %q missing from inventory", scenario.ScenarioID)
		}
		mapped, ok := mapping.LookupScenario(scenario.ScenarioID)
		if !ok {
			t.Fatalf("scope scenario %q missing from terminal mapping", scenario.ScenarioID)
		}
		if mapped.Status != scenario.ExpectedStatus {
			t.Fatalf("scope scenario %q expected status %q, mapping has %q", scenario.ScenarioID, scenario.ExpectedStatus, mapped.Status)
		}
		switch scenario.ComparisonGoal {
		case externalmapping.PrototypeComparisonGoalDirectFit:
			directCount++
		case externalmapping.PrototypeComparisonGoalTranslate:
			translatedCount++
		default:
			t.Fatalf("unexpected comparison goal %q", scenario.ComparisonGoal)
		}
	}
	if directCount != 1 {
		t.Fatalf("directCount = %d, want 1", directCount)
	}
	if translatedCount != 1 {
		t.Fatalf("translatedCount = %d, want 1", translatedCount)
	}
}

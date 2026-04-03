package externalmapping

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadScenarioInventory(paths ...string) (ScenarioInventory, error) {
	path := firstPath(paths...)
	if strings.TrimSpace(path) == "" {
		path = DefaultScenarioInventoryPath
	}
	var inventory ScenarioInventory
	err := readJSONCandidates(&inventory,
		path,
		"scenario_inventory.json",
		filepath.Join("..", "external_mapping", "scenario_inventory.json"),
		filepath.Join("..", "..", "external_mapping", "scenario_inventory.json"),
		filepath.Join("benchmarks", "external_mapping", "scenario_inventory.json"),
	)
	return inventory, err
}

func LoadFamilyMapping(paths ...string) (FamilyMapping, error) {
	path := firstPath(paths...)
	if strings.TrimSpace(path) == "" {
		path = DefaultTerminalBenchMappingPath
	}
	var mapping FamilyMapping
	err := readJSONCandidates(&mapping,
		path,
		filepath.Base(path),
		filepath.Join("..", "external_mapping", filepath.Base(path)),
		filepath.Join("..", "..", "external_mapping", filepath.Base(path)),
		filepath.Join("benchmarks", "external_mapping", filepath.Base(path)),
	)
	return mapping, err
}

func LoadNativeSuiteReport(paths ...string) (NativeSuiteReport, error) {
	path := firstPath(paths...)
	if strings.TrimSpace(path) == "" {
		path = DefaultNativeBaselineReportPath
	}
	var report NativeSuiteReport
	err := readJSONCandidates(&report,
		path,
		filepath.Base(path),
		filepath.Join("benchmarks", "simsh_native_reference", "reports", filepath.Base(path)),
		filepath.Join("..", "simsh_native_reference", "reports", filepath.Base(path)),
		filepath.Join("..", "..", "simsh_native_reference", "reports", filepath.Base(path)),
	)
	return report, err
}

func LoadTerminalBenchPrototypeScope(paths ...string) (TerminalBenchPrototypeScope, error) {
	path := firstPath(paths...)
	if strings.TrimSpace(path) == "" {
		path = DefaultTerminalBenchPrototypeScopePath
	}
	var scope TerminalBenchPrototypeScope
	err := readJSONCandidates(&scope,
		path,
		filepath.Base(path),
		filepath.Join("benchmarks", "terminal_bench_compare", filepath.Base(path)),
		filepath.Join("..", "terminal_bench_compare", filepath.Base(path)),
		filepath.Join("..", "..", "terminal_bench_compare", filepath.Base(path)),
	)
	return scope, err
}

func (inventory ScenarioInventory) LookupScenario(id string) (ScenarioInventoryRecord, bool) {
	for _, scenario := range inventory.Scenarios {
		if scenario.ID == id {
			return scenario, true
		}
	}
	return ScenarioInventoryRecord{}, false
}

func (mapping FamilyMapping) LookupScenario(id string) (FamilyMappingScenario, bool) {
	for _, scenario := range mapping.Scenarios {
		if scenario.ScenarioID == id {
			return scenario, true
		}
	}
	return FamilyMappingScenario{}, false
}

func firstPath(paths ...string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func readJSONCandidates(dest any, candidates ...string) error {
	seen := map[string]struct{}{}
	tried := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(strings.TrimSpace(candidate))
		if candidate == "." || candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		tried = append(tried, candidate)
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, dest); err != nil {
			return fmt.Errorf("parse %s: %w", candidate, err)
		}
		return nil
	}
	return fmt.Errorf("read json candidates failed: %s", strings.Join(tried, ", "))
}

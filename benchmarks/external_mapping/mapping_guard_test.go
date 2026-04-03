package externalmapping

import (
	"reflect"
	"testing"

	benchmarkscenarios "github.com/khicago/simsh/benchmarks/internal/scenarios"
)

func TestScenarioInventoryMatchesNativeCatalog(t *testing.T) {
	t.Parallel()

	inventory, err := LoadScenarioInventory("")
	if err != nil {
		t.Fatalf("LoadScenarioInventory() error = %v", err)
	}
	if !reflect.DeepEqual(inventory.IdentityContract.CanonicalFields, []string{"id", "category"}) {
		t.Errorf("scenario inventory canonical_fields = %#v, want %#v", inventory.IdentityContract.CanonicalFields, []string{"id", "category"})
	}
	if !reflect.DeepEqual(inventory.IdentityContract.CuratedFields, []string{"task_shape", "summary", "truth_surfaces"}) {
		t.Errorf("scenario inventory curated_fields = %#v, want %#v", inventory.IdentityContract.CuratedFields, []string{"task_shape", "summary", "truth_surfaces"})
	}

	records := make([]benchmarkscenarios.InventoryRecord, 0, len(inventory.Scenarios))
	for _, record := range inventory.Scenarios {
		records = append(records, benchmarkscenarios.InventoryRecord{
			Identity: benchmarkscenarios.Identity{
				ID:       record.ID,
				Category: record.Category,
			},
			TaskShape:     record.TaskShape,
			Summary:       record.Summary,
			TruthSurfaces: record.TruthSurfaces,
		})
	}

	want := benchmarkscenarios.NativeReferenceInventory()
	if !reflect.DeepEqual(want, records) {
		t.Errorf("scenario inventory mismatch: got %#v, want %#v", records, want)
	}
}

func TestExternalMappingsCoverEveryNativeScenario(t *testing.T) {
	t.Parallel()

	inventory, err := LoadScenarioInventory("")
	if err != nil {
		t.Fatalf("LoadScenarioInventory() error = %v", err)
	}
	terminalBench, err := LoadFamilyMapping(DefaultTerminalBenchMappingPath)
	if err != nil {
		t.Fatalf("LoadFamilyMapping(Terminal-Bench) error = %v", err)
	}
	sweBenchLive, err := LoadFamilyMapping(DefaultSWEBenchLiveMappingPath)
	if err != nil {
		t.Fatalf("LoadFamilyMapping(SWE-bench-Live) error = %v", err)
	}

	assertScenarioIDs(t, "inventory", inventoryIDs(inventory), benchmarkscenarios.NativeReferenceIDs())
	assertScenarioIDs(t, "terminal_bench_mapping", mappingScenarioIDs(terminalBench), inventoryIDs(inventory))
	assertScenarioIDs(t, "swe_bench_live_mapping", mappingScenarioIDs(sweBenchLive), inventoryIDs(inventory))
}

func TestExternalMappingStatusesAndNotesAreCoherent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		path               string
		wantExternalFamily string
		wantRelationship   string
	}{
		{
			name:               "terminal bench",
			path:               DefaultTerminalBenchMappingPath,
			wantExternalFamily: "Terminal-Bench",
			wantRelationship:   "closest_external_family",
		},
		{
			name:               "swe bench live",
			path:               DefaultSWEBenchLiveMappingPath,
			wantExternalFamily: "SWE-bench-Live",
			wantRelationship:   "dynamic_workload_reference",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := LoadFamilyMapping(tt.path)
			if err != nil {
				t.Fatalf("LoadFamilyMapping(%q) error = %v", tt.path, err)
			}
			if mapping.ExternalFamily != tt.wantExternalFamily || mapping.Relationship != tt.wantRelationship {
				t.Errorf("%s headers = (%q, %q), want (%q, %q)", tt.name, mapping.ExternalFamily, mapping.Relationship, tt.wantExternalFamily, tt.wantRelationship)
			}
			for _, scenario := range mapping.Scenarios {
				switch scenario.Status {
				case MappingStatusAsIs:
					if scenario.TranslationNotes != "" {
						t.Errorf("%s scenario %s translation_notes = %q, want empty for as_is", tt.name, scenario.ScenarioID, scenario.TranslationNotes)
					}
				case MappingStatusTranslated:
					if scenario.TranslationNotes == "" {
						t.Errorf("%s scenario %s translation_notes empty for translated mapping", tt.name, scenario.ScenarioID)
					}
				case MappingStatusExcluded:
					if scenario.TranslationNotes != "" {
						t.Errorf("%s scenario %s translation_notes = %q, want empty for excluded mapping", tt.name, scenario.ScenarioID, scenario.TranslationNotes)
					}
				default:
					t.Errorf("%s scenario %s status = %q, want as_is|translated|excluded", tt.name, scenario.ScenarioID, scenario.Status)
				}
				if scenario.Rationale == "" {
					t.Errorf("%s scenario %s rationale empty", tt.name, scenario.ScenarioID)
				}
			}
		})
	}
}

func inventoryIDs(inventory ScenarioInventory) []string {
	ids := make([]string, 0, len(inventory.Scenarios))
	for _, scenario := range inventory.Scenarios {
		ids = append(ids, scenario.ID)
	}
	return ids
}

func mappingScenarioIDs(mapping FamilyMapping) []string {
	ids := make([]string, 0, len(mapping.Scenarios))
	for _, scenario := range mapping.Scenarios {
		ids = append(ids, scenario.ScenarioID)
	}
	return ids
}

func assertScenarioIDs(t *testing.T, label string, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Errorf("%s scenario ids mismatch: got %#v, want %#v", label, got, want)
	}
}

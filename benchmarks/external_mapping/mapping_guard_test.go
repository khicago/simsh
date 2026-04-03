package externalmapping

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	benchmarkscenarios "github.com/khicago/simsh/benchmarks/internal/scenarios"
)

type scenarioInventory struct {
	IdentityContract struct {
		CanonicalFields []string `json:"canonical_fields"`
		CuratedFields   []string `json:"curated_fields"`
	} `json:"identity_contract"`
	Scenarios []struct {
		ID            string   `json:"id"`
		Category      string   `json:"category"`
		TaskShape     string   `json:"task_shape"`
		Summary       string   `json:"summary"`
		TruthSurfaces []string `json:"truth_surfaces"`
	} `json:"scenarios"`
}

type familyMapping struct {
	ExternalFamily string `json:"external_family"`
	Relationship   string `json:"relationship"`
	Scenarios      []struct {
		ScenarioID       string `json:"scenario_id"`
		Status           string `json:"status"`
		Rationale        string `json:"rationale"`
		TranslationNotes string `json:"translation_notes"`
	} `json:"scenarios"`
}

func TestScenarioInventoryMatchesNativeCatalog(t *testing.T) {
	t.Parallel()

	var inventory scenarioInventory
	readJSON(t, "scenario_inventory.json", &inventory)
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

	inventory := mustScenarioIDs(t, "scenario_inventory.json")
	terminalBench := mustFamilyMapping(t, "terminal_bench_mapping.json")
	sweBenchLive := mustFamilyMapping(t, "swe_bench_live_mapping.json")

	assertScenarioIDs(t, "inventory", inventory, benchmarkscenarios.NativeReferenceIDs())
	assertScenarioIDs(t, "terminal_bench_mapping", terminalBench, inventory)
	assertScenarioIDs(t, "swe_bench_live_mapping", sweBenchLive, inventory)
}

func TestExternalMappingStatusesAndNotesAreCoherent(t *testing.T) {
	t.Parallel()

	for _, fileName := range []string{
		"terminal_bench_mapping.json",
		"swe_bench_live_mapping.json",
	} {
		mapping := mustFamilyMappingWithDetails(t, fileName)
		switch fileName {
		case "terminal_bench_mapping.json":
			if mapping.ExternalFamily != "Terminal-Bench" || mapping.Relationship != "closest_external_family" {
				t.Errorf("%s headers = (%q, %q), want (%q, %q)", fileName, mapping.ExternalFamily, mapping.Relationship, "Terminal-Bench", "closest_external_family")
			}
		case "swe_bench_live_mapping.json":
			if mapping.ExternalFamily != "SWE-bench-Live" || mapping.Relationship != "dynamic_workload_reference" {
				t.Errorf("%s headers = (%q, %q), want (%q, %q)", fileName, mapping.ExternalFamily, mapping.Relationship, "SWE-bench-Live", "dynamic_workload_reference")
			}
		}
		for _, scenario := range mapping.Scenarios {
			switch scenario.Status {
			case "as_is":
				if scenario.TranslationNotes != "" {
					t.Errorf("%s scenario %s translation_notes = %q, want empty for as_is", fileName, scenario.ScenarioID, scenario.TranslationNotes)
				}
			case "translated":
				if scenario.TranslationNotes == "" {
					t.Errorf("%s scenario %s translation_notes empty for translated mapping", fileName, scenario.ScenarioID)
				}
			case "excluded":
				if scenario.TranslationNotes != "" {
					t.Errorf("%s scenario %s translation_notes = %q, want empty for excluded mapping", fileName, scenario.ScenarioID, scenario.TranslationNotes)
				}
			default:
				t.Errorf("%s scenario %s status = %q, want as_is|translated|excluded", fileName, scenario.ScenarioID, scenario.Status)
			}
			if scenario.Rationale == "" {
				t.Errorf("%s scenario %s rationale empty", fileName, scenario.ScenarioID)
			}
		}
	}
}

func mustScenarioIDs(t *testing.T, fileName string) []string {
	t.Helper()

	var inventory scenarioInventory
	readJSON(t, fileName, &inventory)
	ids := make([]string, 0, len(inventory.Scenarios))
	for _, scenario := range inventory.Scenarios {
		ids = append(ids, scenario.ID)
	}
	return ids
}

func mustFamilyMapping(t *testing.T, fileName string) []string {
	t.Helper()

	mapping := mustFamilyMappingWithDetails(t, fileName)
	ids := make([]string, 0, len(mapping.Scenarios))
	for _, scenario := range mapping.Scenarios {
		ids = append(ids, scenario.ScenarioID)
	}
	return ids
}

func mustFamilyMappingWithDetails(t *testing.T, fileName string) familyMapping {
	t.Helper()

	var mapping familyMapping
	readJSON(t, fileName, &mapping)
	return mapping
}

func assertScenarioIDs(t *testing.T, label string, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Errorf("%s scenario ids mismatch: got %#v, want %#v", label, got, want)
	}
}

func readJSON(t *testing.T, fileName string, dest any) {
	t.Helper()

	candidates := []string{
		fileName,
		filepath.Join("benchmarks", "external_mapping", fileName),
	}

	var (
		data []byte
		err  error
	)
	for _, path := range candidates {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("read %s: %v", fileName, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("parse %s: %v", fileName, err)
	}
}

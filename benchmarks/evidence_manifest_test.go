package benchmarks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
)

const evidenceManifestPath = "benchmarks/evidence_manifest.json"

type evidenceManifest struct {
	Version                     int    `json:"version"`
	BenchmarkRefreshCommand     string `json:"benchmark_refresh_command"`
	PairedUpliftRefreshCommand  string `json:"paired_uplift_refresh_command"`
	NativeBaseline              struct {
		ReportPath string   `json:"report_path"`
		Volatile   []string `json:"volatile_fields"`
	} `json:"native_baseline"`
	TerminalBenchPrototype struct {
		ScopePath             string `json:"scope_path"`
		ArtifactJSONPath      string `json:"artifact_json_path"`
		ArtifactMDPath        string `json:"artifact_md_path"`
		SourceReportPathField string `json:"source_report_path_field"`
	} `json:"terminal_bench_prototype"`
	PairedUplift struct {
		TaskManifestPath      string `json:"task_manifest_path"`
		RawSnapshotPath       string `json:"raw_snapshot_path"`
		ArtifactJSONPath      string `json:"artifact_json_path"`
		ArtifactMDPath        string `json:"artifact_md_path"`
		FailureTaxonomyPath   string `json:"failure_taxonomy_path"`
		SourceSnapshotField   string `json:"source_snapshot_path_field"`
		TaskManifestPathField string `json:"task_manifest_path_field"`
	} `json:"paired_uplift"`
}

func TestEvidenceManifestMatchesCheckedInProofArtifacts(t *testing.T) {
	t.Parallel()

	manifest := loadEvidenceManifest(t)
	if manifest.Version != 1 {
		t.Fatalf("manifest.Version = %d, want 1", manifest.Version)
	}
	if manifest.BenchmarkRefreshCommand != "make benchmark-refresh" {
		t.Fatalf("manifest.BenchmarkRefreshCommand = %q, want make benchmark-refresh", manifest.BenchmarkRefreshCommand)
	}
	if manifest.PairedUpliftRefreshCommand != "make benchmark-uplift" {
		t.Fatalf("manifest.PairedUpliftRefreshCommand = %q, want make benchmark-uplift", manifest.PairedUpliftRefreshCommand)
	}
	assertMakeTargetExists(t, "benchmark-refresh")
	assertMakeTargetExists(t, "benchmark-uplift")

	if manifest.NativeBaseline.ReportPath != externalmapping.DefaultNativeBaselineReportPath {
		t.Fatalf("native baseline path = %q, want %q", manifest.NativeBaseline.ReportPath, externalmapping.DefaultNativeBaselineReportPath)
	}
	if manifest.TerminalBenchPrototype.ScopePath != externalmapping.DefaultTerminalBenchPrototypeScopePath {
		t.Fatalf("terminal scope path = %q, want %q", manifest.TerminalBenchPrototype.ScopePath, externalmapping.DefaultTerminalBenchPrototypeScopePath)
	}
	if manifest.TerminalBenchPrototype.ArtifactJSONPath != externalmapping.DefaultTerminalBenchArtifactPath {
		t.Fatalf("terminal artifact json path = %q, want %q", manifest.TerminalBenchPrototype.ArtifactJSONPath, externalmapping.DefaultTerminalBenchArtifactPath)
	}
	if manifest.TerminalBenchPrototype.ArtifactMDPath != externalmapping.DefaultTerminalBenchSummaryPath {
		t.Fatalf("terminal artifact md path = %q, want %q", manifest.TerminalBenchPrototype.ArtifactMDPath, externalmapping.DefaultTerminalBenchSummaryPath)
	}

	assertFileExists(t, manifest.NativeBaseline.ReportPath)
	assertFileExists(t, manifest.TerminalBenchPrototype.ScopePath)
	assertFileExists(t, manifest.TerminalBenchPrototype.ArtifactJSONPath)
	assertFileExists(t, manifest.TerminalBenchPrototype.ArtifactMDPath)
	assertFileExists(t, manifest.PairedUplift.TaskManifestPath)
	assertFileExists(t, manifest.PairedUplift.RawSnapshotPath)
	assertFileExists(t, manifest.PairedUplift.ArtifactJSONPath)
	assertFileExists(t, manifest.PairedUplift.ArtifactMDPath)
	assertFileExists(t, manifest.PairedUplift.FailureTaxonomyPath)

	terminalArtifact := loadTerminalArtifact(t, manifest.TerminalBenchPrototype.ArtifactJSONPath)
	if manifest.TerminalBenchPrototype.SourceReportPathField != "source.report_path" {
		t.Fatalf("terminal source report field = %q, want source.report_path", manifest.TerminalBenchPrototype.SourceReportPathField)
	}
	if terminalArtifact.Source.ReportPath != manifest.NativeBaseline.ReportPath {
		t.Fatalf("terminal artifact source report = %q, want %q", terminalArtifact.Source.ReportPath, manifest.NativeBaseline.ReportPath)
	}

	pairedArtifact := loadPairedArtifact(t, manifest.PairedUplift.ArtifactJSONPath)
	if manifest.PairedUplift.TaskManifestPathField != "task_manifest_path" {
		t.Fatalf("paired task manifest field = %q, want task_manifest_path", manifest.PairedUplift.TaskManifestPathField)
	}
	if pairedArtifact.TaskManifestPath != manifest.PairedUplift.TaskManifestPath {
		t.Fatalf("paired artifact task manifest = %q, want %q", pairedArtifact.TaskManifestPath, manifest.PairedUplift.TaskManifestPath)
	}

	failureTaxonomy := loadFailureTaxonomy(t, manifest.PairedUplift.FailureTaxonomyPath)
	if manifest.PairedUplift.SourceSnapshotField != "source_snapshot_path" {
		t.Fatalf("paired source snapshot field = %q, want source_snapshot_path", manifest.PairedUplift.SourceSnapshotField)
	}
	if failureTaxonomy.SourceSnapshotPath != manifest.PairedUplift.RawSnapshotPath {
		t.Fatalf("failure taxonomy source snapshot = %q, want %q", failureTaxonomy.SourceSnapshotPath, manifest.PairedUplift.RawSnapshotPath)
	}
}

func loadEvidenceManifest(t *testing.T) evidenceManifest {
	t.Helper()
	raw := readCandidateFile(t, evidenceManifestPath)
	var manifest evidenceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("json.Unmarshal(%q) failed: %v", evidenceManifestPath, err)
	}
	return manifest
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(resolveCandidatePath(t, path)); err != nil {
		t.Fatalf("Stat(%q) failed: %v", path, err)
	}
}

func loadTerminalArtifact(t *testing.T, path string) externalmapping.TerminalBenchComparisonArtifact {
	t.Helper()
	raw := readCandidateFile(t, path)
	var artifact externalmapping.TerminalBenchComparisonArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("json.Unmarshal(%q) failed: %v", path, err)
	}
	return artifact
}

type pairedArtifact struct {
	TaskManifestPath string `json:"task_manifest_path"`
}

func loadPairedArtifact(t *testing.T, path string) pairedArtifact {
	t.Helper()
	raw := readCandidateFile(t, path)
	var artifact pairedArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("json.Unmarshal(%q) failed: %v", path, err)
	}
	return artifact
}

type failureTaxonomyReport struct {
	SourceSnapshotPath string `json:"source_snapshot_path"`
}

func loadFailureTaxonomy(t *testing.T, path string) failureTaxonomyReport {
	t.Helper()
	raw := readCandidateFile(t, path)
	var report failureTaxonomyReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json.Unmarshal(%q) failed: %v", path, err)
	}
	return report
}

func readCandidateFile(t *testing.T, path string) []byte {
	t.Helper()
	resolved := resolveCandidatePath(t, path)
	raw, err := os.ReadFile(resolved)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", path, err)
	}
	return raw
}

func resolveCandidatePath(t *testing.T, path string) string {
	t.Helper()
	candidates := []string{
		path,
		filepath.Base(path),
		filepath.Join("benchmarks", filepath.Base(path)),
		filepath.Join("..", path),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatalf("unable to resolve path %q from test working directory", path)
	return ""
}

func assertMakeTargetExists(t *testing.T, target string) {
	t.Helper()
	raw := readCandidateFile(t, "Makefile")
	needle := "\n" + target + ":"
	if !strings.Contains("\n"+string(raw), needle) {
		t.Fatalf("Makefile missing target %q", target)
	}
}

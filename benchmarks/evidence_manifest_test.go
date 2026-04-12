package benchmarks_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
)

const evidenceManifestPath = "benchmarks/evidence_manifest.json"

type evidenceManifest struct {
	Version                    int    `json:"version"`
	BenchmarkRefreshCommand    string `json:"benchmark_refresh_command"`
	PairedUpliftRefreshCommand string `json:"paired_uplift_refresh_command"`
	NativeBaseline             struct {
		ReportPath   string   `json:"report_path"`
		ReportSHA256 string   `json:"report_sha256"`
		Volatile     []string `json:"volatile_fields"`
	} `json:"native_baseline"`
	TerminalBenchPrototype struct {
		ScopePath             string `json:"scope_path"`
		ScopeSHA256           string `json:"scope_sha256"`
		ArtifactJSONPath      string `json:"artifact_json_path"`
		ArtifactJSONSHA256    string `json:"artifact_json_sha256"`
		ArtifactMDPath        string `json:"artifact_md_path"`
		ArtifactMDSHA256      string `json:"artifact_md_sha256"`
		SourceReportPathField string `json:"source_report_path_field"`
	} `json:"terminal_bench_prototype"`
	PairedUplift struct {
		TaskManifestPath      string               `json:"task_manifest_path"`
		TaskManifestSHA256    string               `json:"task_manifest_sha256"`
		RawSnapshotPath       string               `json:"raw_snapshot_path"`
		RawSnapshotSHA256     string               `json:"raw_snapshot_sha256"`
		ArtifactJSONPath      string               `json:"artifact_json_path"`
		ArtifactJSONSHA256    string               `json:"artifact_json_sha256"`
		ArtifactMDPath        string               `json:"artifact_md_path"`
		ArtifactMDSHA256      string               `json:"artifact_md_sha256"`
		FailureTaxonomyPath   string               `json:"failure_taxonomy_path"`
		FailureTaxonomySHA256 string               `json:"failure_taxonomy_sha256"`
		SourceSnapshotField   string               `json:"source_snapshot_path_field"`
		TaskManifestPathField string               `json:"task_manifest_path_field"`
		DownstreamArtifacts   []downstreamArtifact `json:"downstream_artifacts"`
		Volatile              []string             `json:"volatile_fields"`
	} `json:"paired_uplift"`
}

type downstreamArtifact struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	DerivedFrom string `json:"derived_from"`
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
	assertMakeTargetCommand(t, "benchmark-refresh", "$(GO) run ./benchmarks/refresh_terminal_bench_compare")
	assertMakeTargetCommand(t, "benchmark-uplift", "$(GO) run ./benchmarks/paired_uplift")

	if manifest.NativeBaseline.ReportPath != externalmapping.DefaultNativeBaselineReportPath {
		t.Fatalf("native baseline path = %q, want %q", manifest.NativeBaseline.ReportPath, externalmapping.DefaultNativeBaselineReportPath)
	}
	assertSHA256Matches(t, manifest.NativeBaseline.ReportPath, manifest.NativeBaseline.ReportSHA256)
	assertVolatileFields(t, "native baseline", manifest.NativeBaseline.Volatile, []string{
		"generated_at",
		"scenarios[*].duration_ms",
	})
	if manifest.TerminalBenchPrototype.ScopePath != externalmapping.DefaultTerminalBenchPrototypeScopePath {
		t.Fatalf("terminal scope path = %q, want %q", manifest.TerminalBenchPrototype.ScopePath, externalmapping.DefaultTerminalBenchPrototypeScopePath)
	}
	assertSHA256Matches(t, manifest.TerminalBenchPrototype.ScopePath, manifest.TerminalBenchPrototype.ScopeSHA256)
	if manifest.TerminalBenchPrototype.ArtifactJSONPath != externalmapping.DefaultTerminalBenchArtifactPath {
		t.Fatalf("terminal artifact json path = %q, want %q", manifest.TerminalBenchPrototype.ArtifactJSONPath, externalmapping.DefaultTerminalBenchArtifactPath)
	}
	assertSHA256Matches(t, manifest.TerminalBenchPrototype.ArtifactJSONPath, manifest.TerminalBenchPrototype.ArtifactJSONSHA256)
	if manifest.TerminalBenchPrototype.ArtifactMDPath != externalmapping.DefaultTerminalBenchSummaryPath {
		t.Fatalf("terminal artifact md path = %q, want %q", manifest.TerminalBenchPrototype.ArtifactMDPath, externalmapping.DefaultTerminalBenchSummaryPath)
	}
	assertSHA256Matches(t, manifest.TerminalBenchPrototype.ArtifactMDPath, manifest.TerminalBenchPrototype.ArtifactMDSHA256)

	assertFileExists(t, manifest.NativeBaseline.ReportPath)
	assertFileExists(t, manifest.TerminalBenchPrototype.ScopePath)
	assertFileExists(t, manifest.TerminalBenchPrototype.ArtifactJSONPath)
	assertFileExists(t, manifest.TerminalBenchPrototype.ArtifactMDPath)
	assertFileExists(t, manifest.PairedUplift.TaskManifestPath)
	assertFileExists(t, manifest.PairedUplift.RawSnapshotPath)
	assertFileExists(t, manifest.PairedUplift.ArtifactJSONPath)
	assertFileExists(t, manifest.PairedUplift.ArtifactMDPath)
	assertFileExists(t, manifest.PairedUplift.FailureTaxonomyPath)
	assertSHA256Matches(t, manifest.PairedUplift.TaskManifestPath, manifest.PairedUplift.TaskManifestSHA256)
	assertSHA256Matches(t, manifest.PairedUplift.RawSnapshotPath, manifest.PairedUplift.RawSnapshotSHA256)
	assertSHA256Matches(t, manifest.PairedUplift.ArtifactJSONPath, manifest.PairedUplift.ArtifactJSONSHA256)
	assertSHA256Matches(t, manifest.PairedUplift.ArtifactMDPath, manifest.PairedUplift.ArtifactMDSHA256)
	assertSHA256Matches(t, manifest.PairedUplift.FailureTaxonomyPath, manifest.PairedUplift.FailureTaxonomySHA256)
	assertVolatileFields(t, "paired uplift", manifest.PairedUplift.Volatile, []string{
		"raw_snapshot.generated_at",
		"raw_snapshot.tasks[*].simsh.duration_ms",
		"raw_snapshot.tasks[*].baseline.duration_ms",
		"artifact_json.generated_at",
		"artifact_json.tasks[*].simsh.duration_ms",
		"artifact_json.tasks[*].baseline.duration_ms",
		"artifact_json.tasks[*].delta.duration_ms_delta",
		"failure_taxonomy.generated_at",
	})
	assertPairedDownstreamArtifacts(t, manifest.PairedUplift.DownstreamArtifacts, map[string]string{
		"aggregate_json":        manifest.PairedUplift.ArtifactJSONPath,
		"summary_md":            manifest.PairedUplift.ArtifactMDPath,
		"failure_taxonomy_json": manifest.PairedUplift.FailureTaxonomyPath,
	}, manifest.PairedUplift.RawSnapshotPath)

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

func assertSHA256Matches(t *testing.T, path, want string) {
	t.Helper()
	if want == "" {
		t.Fatalf("manifest sha256 for %q is empty", path)
	}
	raw := readCandidateFile(t, path)
	got := fmt.Sprintf("%x", sha256.Sum256(raw))
	if got != want {
		t.Fatalf("sha256(%q) = %s, want %s", path, got, want)
	}
}

func assertVolatileFields(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s volatile fields = %v, want %v", name, got, want)
	}
}

func assertPairedDownstreamArtifacts(t *testing.T, got []downstreamArtifact, wantByKind map[string]string, wantSource string) {
	t.Helper()
	if len(got) != len(wantByKind) {
		t.Fatalf("paired downstream artifacts length = %d, want %d", len(got), len(wantByKind))
	}
	seen := make(map[string]bool, len(got))
	for _, artifact := range got {
		wantPath, ok := wantByKind[artifact.Kind]
		if !ok {
			t.Fatalf("paired downstream artifact kind %q is not expected", artifact.Kind)
		}
		if seen[artifact.Kind] {
			t.Fatalf("paired downstream artifact kind %q appears more than once", artifact.Kind)
		}
		seen[artifact.Kind] = true
		if artifact.Path != wantPath {
			t.Fatalf("paired downstream artifact %q path = %q, want %q", artifact.Kind, artifact.Path, wantPath)
		}
		if artifact.DerivedFrom != wantSource {
			t.Fatalf("paired downstream artifact %q source = %q, want %q", artifact.Kind, artifact.DerivedFrom, wantSource)
		}
		assertFileExists(t, artifact.Path)
	}
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

func assertMakeTargetCommand(t *testing.T, target, command string) {
	t.Helper()
	raw := readCandidateFile(t, "Makefile")
	needle := "\n" + target + ":"
	if !strings.Contains("\n"+string(raw), needle) {
		t.Fatalf("Makefile missing target %q", target)
	}
	if !strings.Contains(string(raw), needle+"\n\t"+command) {
		t.Fatalf("Makefile target %q does not contain command %q", target, command)
	}
}

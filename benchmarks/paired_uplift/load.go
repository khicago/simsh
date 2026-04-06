package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadTaskManifest(paths ...string) (TaskManifest, error) {
	path := firstCandidatePath(paths...)
	if strings.TrimSpace(path) == "" {
		path = DefaultTaskManifestPath
	}
	var manifest TaskManifest
	err := readJSONCandidates(&manifest,
		path,
		"task_set.json",
		filepath.Join("benchmarks", "paired_uplift", "task_set.json"),
		filepath.Join("..", "paired_uplift", "task_set.json"),
		filepath.Join("..", "..", "paired_uplift", "task_set.json"),
	)
	return manifest, err
}

func LoadPairedRunSnapshot(paths ...string) (PairedRunSnapshot, error) {
	path := firstCandidatePath(paths...)
	if strings.TrimSpace(path) == "" {
		path = DefaultSnapshotPath
	}
	var snapshot PairedRunSnapshot
	err := readJSONCandidates(&snapshot,
		path,
		filepath.Base(path),
		filepath.Join("benchmarks", "paired_uplift", "reports", filepath.Base(path)),
		filepath.Join("..", "paired_uplift", "reports", filepath.Base(path)),
		filepath.Join("..", "..", "paired_uplift", "reports", filepath.Base(path)),
	)
	return snapshot, err
}

func MarshalPairedRunSnapshotJSON(snapshot PairedRunSnapshot) ([]byte, error) {
	return json.MarshalIndent(snapshot, "", "  ")
}

func MarshalPairedUpliftArtifactJSON(artifact PairedUpliftArtifact) ([]byte, error) {
	return json.MarshalIndent(artifact, "", "  ")
}

func MarshalFailureTaxonomyJSON(report FailureTaxonomyReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func firstCandidatePath(paths ...string) string {
	if len(paths) == 0 {
		return ""
	}
	return strings.TrimSpace(paths[0])
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

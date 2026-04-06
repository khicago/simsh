package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
)

func RunPairedUplift(ctx context.Context, manifest TaskManifest, manifestPath string) (PairedRunSnapshot, error) {
	inventory, err := externalmapping.LoadScenarioInventory("")
	if err != nil {
		return PairedRunSnapshot{}, err
	}
	if err := validateTaskManifest(manifest, inventory); err != nil {
		return PairedRunSnapshot{}, err
	}
	snapshot := PairedRunSnapshot{
		Version:           1,
		GeneratedAt:       time.Now().UTC(),
		ComparisonRule:    manifest.ComparisonRule,
		TaskManifestPath:  manifestPath,
		AgentID:           manifest.AgentID,
		BaselineSubstrate: manifest.BaselineSubstrate,
		SimshSubstrate:    manifest.SimshSubstrate,
		Tasks:             make([]PairRunRecord, 0, len(manifest.Tasks)),
	}
	for _, task := range manifest.Tasks {
		record, err := runTaskPair(ctx, inventory, manifest, task, manifestPath)
		if err != nil {
			return PairedRunSnapshot{}, err
		}
		snapshot.Tasks = append(snapshot.Tasks, record)
	}
	return snapshot, nil
}

func BuildPairedUpliftArtifact(snapshot PairedRunSnapshot) (PairedUpliftArtifact, error) {
	if snapshot.ComparisonRule != pairedUpliftComparisonRule {
		return PairedUpliftArtifact{}, fmt.Errorf("unexpected snapshot comparison rule %q", snapshot.ComparisonRule)
	}
	artifact := PairedUpliftArtifact{
		Version:           1,
		GeneratedAt:       snapshot.GeneratedAt,
		ComparisonRule:    snapshot.ComparisonRule,
		TaskManifestPath:  snapshot.TaskManifestPath,
		AgentID:           snapshot.AgentID,
		BaselineSubstrate: snapshot.BaselineSubstrate,
		SimshSubstrate:    snapshot.SimshSubstrate,
		Tasks:             make([]ComparedTask, 0, len(snapshot.Tasks)),
	}
	for _, task := range snapshot.Tasks {
		compared := ComparedTask{
			ScenarioID:      task.ScenarioID,
			Category:        task.Category,
			TaskShape:       task.TaskShape,
			Summary:         task.Summary,
			TruthSurfaces:   append([]string(nil), task.TruthSurfaces...),
			PairSeed:        task.PairSeed,
			RunOrder:        task.RunOrder,
			AgentID:         task.AgentID,
			Budget:          task.Budget,
			ExpectedOutputs: append([]string(nil), task.ExpectedOutputs...),
			WhySelected:     task.WhySelected,
			EvidenceRefs:    append([]string(nil), task.EvidenceRefs...),
			Simsh:           task.Simsh,
			Baseline:        task.Baseline,
		}
		compared.Delta = PairDelta{
			SuccessDelta:                 boolDelta(task.Simsh.Success, task.Baseline.Success),
			RetryDelta:                   task.Baseline.Retries - task.Simsh.Retries,
			WastedStepDelta:              task.Baseline.WastedSteps - task.Simsh.WastedSteps,
			WastedObservationTokensDelta: task.Baseline.WastedObservationTokens - task.Simsh.WastedObservationTokens,
			MisunderstandingDelta:        task.Baseline.EnvironmentMisunderstandings - task.Simsh.EnvironmentMisunderstandings,
			DurationMSDelta:              task.Baseline.DurationMS - task.Simsh.DurationMS,
		}
		compared.Winner = compareTaskWinner(task.Simsh, task.Baseline)
		artifact.Tasks = append(artifact.Tasks, compared)

		artifact.Summary.TotalTasks++
		if task.Simsh.Success {
			artifact.Summary.SimshSuccessCount++
		}
		if task.Baseline.Success {
			artifact.Summary.BaselineSuccessCount++
		}
		artifact.Summary.SimshRetries += task.Simsh.Retries
		artifact.Summary.BaselineRetries += task.Baseline.Retries
		artifact.Summary.SimshWastedSteps += task.Simsh.WastedSteps
		artifact.Summary.BaselineWastedSteps += task.Baseline.WastedSteps
		artifact.Summary.SimshMisunderstandings += task.Simsh.EnvironmentMisunderstandings
		artifact.Summary.BaselineMisunderstandings += task.Baseline.EnvironmentMisunderstandings
		artifact.Summary.SimshObservationTokens += task.Simsh.ApproxObservationTokens
		artifact.Summary.BaselineObservationTokens += task.Baseline.ApproxObservationTokens
		artifact.Summary.SimshWastedObservationTokens += task.Simsh.WastedObservationTokens
		artifact.Summary.BaselineWastedObservationTokens += task.Baseline.WastedObservationTokens
	}
	return artifact, nil
}

func BuildFailureTaxonomy(snapshot PairedRunSnapshot, snapshotPath string) FailureTaxonomyReport {
	type key struct {
		bucket  string
		runtime string
		kind    string
	}
	type value struct {
		count       int
		scenarioSet map[string]struct{}
	}
	rollup := map[key]*value{}
	add := func(bucket, runtime, kind, scenarioID string) {
		if strings.TrimSpace(kind) == "" {
			return
		}
		k := key{bucket: bucket, runtime: runtime, kind: kind}
		entry, ok := rollup[k]
		if !ok {
			entry = &value{scenarioSet: map[string]struct{}{}}
			rollup[k] = entry
		}
		entry.count++
		entry.scenarioSet[scenarioID] = struct{}{}
	}
	for _, task := range snapshot.Tasks {
		if task.Simsh.FailureKind != "" {
			add(taxonomyBucketFailure, task.Simsh.Substrate, task.Simsh.FailureKind, task.ScenarioID)
		}
		if task.Baseline.FailureKind != "" {
			add(taxonomyBucketFailure, task.Baseline.Substrate, task.Baseline.FailureKind, task.ScenarioID)
		}
		for _, step := range task.Simsh.StepsDetail {
			if step.EnvironmentMisunderstood {
				add(taxonomyBucketMisunderstanding, task.Simsh.Substrate, step.MisunderstandingKind, task.ScenarioID)
			}
		}
		for _, step := range task.Baseline.StepsDetail {
			if step.EnvironmentMisunderstood {
				add(taxonomyBucketMisunderstanding, task.Baseline.Substrate, step.MisunderstandingKind, task.ScenarioID)
			}
		}
	}
	entries := make([]FailureTaxonomyEntry, 0, len(rollup))
	for k, v := range rollup {
		scenarioIDs := make([]string, 0, len(v.scenarioSet))
		for scenarioID := range v.scenarioSet {
			scenarioIDs = append(scenarioIDs, scenarioID)
		}
		sort.Strings(scenarioIDs)
		entries = append(entries, FailureTaxonomyEntry{
			Bucket:      k.bucket,
			Runtime:     k.runtime,
			Kind:        k.kind,
			Count:       v.count,
			ScenarioIDs: scenarioIDs,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		return left.Bucket+left.Runtime+left.Kind < right.Bucket+right.Runtime+right.Kind
	})
	return FailureTaxonomyReport{
		Version:            1,
		GeneratedAt:        snapshot.GeneratedAt,
		SourceSnapshotPath: snapshotPath,
		Entries:            entries,
	}
}

func compareTaskWinner(simsh, baseline SubstrateRunRecord) string {
	if simsh.Success != baseline.Success {
		if simsh.Success {
			return "simsh"
		}
		return "baseline"
	}
	if simsh.Retries != baseline.Retries {
		if simsh.Retries < baseline.Retries {
			return "simsh"
		}
		return "baseline"
	}
	if simsh.WastedObservationTokens != baseline.WastedObservationTokens {
		if simsh.WastedObservationTokens < baseline.WastedObservationTokens {
			return "simsh"
		}
		return "baseline"
	}
	if simsh.EnvironmentMisunderstandings != baseline.EnvironmentMisunderstandings {
		if simsh.EnvironmentMisunderstandings < baseline.EnvironmentMisunderstandings {
			return "simsh"
		}
		return "baseline"
	}
	if simsh.DurationMS != baseline.DurationMS {
		if simsh.DurationMS < baseline.DurationMS {
			return "simsh"
		}
		return "baseline"
	}
	return "tie"
}

func boolDelta(simsh, baseline bool) int {
	switch {
	case simsh && !baseline:
		return 1
	case !simsh && baseline:
		return -1
	default:
		return 0
	}
}

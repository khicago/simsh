package main

import (
	"fmt"
	"strings"
)

func RenderPairedUpliftMarkdown(artifact PairedUpliftArtifact, taxonomy FailureTaxonomyReport) string {
	lines := []string{
		"# Paired A/B Uplift Proof",
		"",
		"## Scope",
		"",
		fmt.Sprintf("- Comparison rule: `%s`", artifact.ComparisonRule),
		fmt.Sprintf("- Agent: `%s`", artifact.AgentID),
		fmt.Sprintf("- simsh substrate: `%s`", artifact.SimshSubstrate),
		fmt.Sprintf("- baseline substrate: `%s`", artifact.BaselineSubstrate),
		fmt.Sprintf("- Task manifest: `%s`", artifact.TaskManifestPath),
		"",
		"## Summary",
		"",
		fmt.Sprintf("- Total tasks: %d", artifact.Summary.TotalTasks),
		fmt.Sprintf("- simsh success count: %d", artifact.Summary.SimshSuccessCount),
		fmt.Sprintf("- baseline success count: %d", artifact.Summary.BaselineSuccessCount),
		fmt.Sprintf("- simsh retries: %d", artifact.Summary.SimshRetries),
		fmt.Sprintf("- baseline retries: %d", artifact.Summary.BaselineRetries),
		fmt.Sprintf("- simsh wasted observation tokens: %d", artifact.Summary.SimshWastedObservationTokens),
		fmt.Sprintf("- baseline wasted observation tokens: %d", artifact.Summary.BaselineWastedObservationTokens),
		"",
		"## Task Results",
		"",
		"| scenario | winner | simsh success | baseline success | retry delta | wasted-token delta | misunderstanding delta |",
		"| --- | --- | --- | --- | --- | --- | --- |",
	}
	for _, task := range artifact.Tasks {
		lines = append(lines, fmt.Sprintf("| `%s` | `%s` | `%t` | `%t` | `%d` | `%d` | `%d` |",
			task.ScenarioID,
			task.Winner,
			task.Simsh.Success,
			task.Baseline.Success,
			task.Delta.RetryDelta,
			task.Delta.WastedObservationTokensDelta,
			task.Delta.MisunderstandingDelta,
		))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Why `%s` is included: %s", task.ScenarioID, task.WhySelected))
		lines = append(lines, fmt.Sprintf("Truth surfaces for `%s`: `%s`", task.ScenarioID, strings.Join(task.TruthSurfaces, "`, `")))
	}
	lines = append(lines,
		"",
		"## Failure Taxonomy",
		"",
		"| bucket | runtime | kind | count | scenarios |",
		"| --- | --- | --- | --- | --- |",
	)
	for _, entry := range taxonomy.Entries {
		lines = append(lines, fmt.Sprintf("| `%s` | `%s` | `%s` | %d | `%s` |",
			entry.Bucket,
			entry.Runtime,
			entry.Kind,
			entry.Count,
			strings.Join(entry.ScenarioIDs, "`, `"),
		))
	}
	lines = append(lines,
		"",
		"## Notes",
		"",
		"- This harness stays downstream from the native benchmark identity contract.",
		"- The baseline is repo-controlled and intentionally thinner; it is not an ambient host-shell bakeoff.",
		"- Raw paired runs are freshness snapshots; this markdown is a deterministic downstream rendering of the checked-in snapshot.",
	)
	return strings.Join(lines, "\n") + "\n"
}

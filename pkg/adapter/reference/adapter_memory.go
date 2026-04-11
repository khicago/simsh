package reference

import (
	"encoding/json"
	"fmt"
	"strings"
)

func memoryViewFiles(state sessionState, raw []byte, docs []projectionRecord, resources []projectionRecord, skills []projectionRecord, curated []curatedRecord, events []controlPlaneEvent) map[string]string {
	files := map[string]string{
		virtualPath("memory", "observations.md"):       strings.Join(state.Observations, "\n"),
		virtualPath("memory", "status.json"):           string(raw),
		virtualPath("memory", "summary.md"):            renderMemorySummary(state, docs, resources, skills, len(curated), events),
		virtualPath("memory", "projection_metrics.md"): renderProjectionMetricsSummary(state, docs, resources, skills, events),
		virtualPath("memory", "denials.md"):            renderDenialSummary(state),
		virtualPath("memory", "workflows.md"):          renderWorkflowSummary(state.Workflows),
		virtualPath("memory", "projections.md"):        renderProjectionSummary(docs, resources, skills),
		virtualPath("memory", "curated.md"):            renderCuratedSummary(curated),
		virtualPath("memory", "skills_audit.md"):       renderControlPlaneEventSummary(events, state.ProjectionGeneration),
	}
	workflowRaw, err := json.MarshalIndent(state.Workflows, "", "  ")
	if err == nil {
		files[virtualPath("memory", "workflows.json")] = string(workflowRaw)
	}
	projectionRaw, err := json.MarshalIndent(projectionView{Documents: docs, Resources: resources, Skills: skills}, "", "  ")
	if err == nil {
		files[virtualPath("memory", "projections.json")] = string(projectionRaw)
	}
	projectionMetricsRaw, err := json.MarshalIndent(buildProjectionMetricsView(state, docs, resources, skills, events), "", "  ")
	if err == nil {
		files[virtualPath("memory", "projection_metrics.json")] = string(projectionMetricsRaw)
	}
	curatedRaw, err := json.MarshalIndent(curated, "", "  ")
	if err == nil {
		files[virtualPath("memory", "curated.json")] = string(curatedRaw)
	}
	denialsRaw, err := json.MarshalIndent(buildDenialView(state), "", "  ")
	if err == nil {
		files[virtualPath("memory", "denials.json")] = string(denialsRaw)
	}
	controlPlaneRaw, err := json.MarshalIndent(controlPlaneEventViews(events, state.ProjectionGeneration), "", "  ")
	if err == nil {
		files[virtualPath("memory", "skills_audit.json")] = string(controlPlaneRaw)
	}
	if len(state.ReadResources) > 0 {
		files[virtualPath("memory", "resources.md")] = strings.Join(state.ReadResources, "\n")
	}
	if len(state.ReadSkills) > 0 {
		files[virtualPath("memory", "skills.md")] = strings.Join(state.ReadSkills, "\n")
	}
	return files
}

func renderMemorySummary(state sessionState, docs []projectionRecord, resources []projectionRecord, skills []projectionRecord, curatedCount int, events []controlPlaneEvent) string {
	freshnessCounts := countProjectionFreshness(docs, resources, skills)
	materializationCounts := countProjectionMaterialization(docs, resources, skills)
	lines := []string{
		"# Managed Memory",
		"",
		fmt.Sprintf("- freshness: %s", state.Freshness),
		fmt.Sprintf("- projection_generation: %d", state.ProjectionGeneration),
		fmt.Sprintf("- projection_build_ms: %d", state.ProjectionBuildMS),
		fmt.Sprintf("- projections.documents: %d", len(docs)),
		fmt.Sprintf("- projections.resources: %d", len(resources)),
		fmt.Sprintf("- projections.skills: %d", len(skills)),
		fmt.Sprintf("- curated_entries: %d", curatedCount),
		fmt.Sprintf("- observations: %d", len(state.Observations)),
		fmt.Sprintf("- reference_reads: %d", len(state.ReadRefs)),
		fmt.Sprintf("- resource_reads: %d", len(state.ReadResources)),
		fmt.Sprintf("- skill_reads: %d", len(state.ReadSkills)),
		fmt.Sprintf("- written_outputs: %d", len(state.WrittenOutputs)),
		fmt.Sprintf("- denied_paths: %d", len(state.DeniedPaths)),
		fmt.Sprintf("- control_plane_events: %d", len(events)),
	}
	if len(events) > 0 {
		last := events[len(events)-1]
		lines = append(lines, fmt.Sprintf("- last_control_plane_event: %s %s", last.Op, last.Path))
	}
	if len(freshnessCounts) > 0 {
		lines = append(lines, "", "## Projection Freshness")
		for _, freshness := range []string{
			projectionFreshnessLive,
			projectionFreshnessSnapshot,
			projectionFreshnessStale,
			projectionFreshnessUpdated,
			projectionFreshnessGenerated,
		} {
			if count := freshnessCounts[freshness]; count > 0 {
				lines = append(lines, fmt.Sprintf("- %s: %d", freshness, count))
			}
		}
	}
	if len(materializationCounts) > 0 {
		lines = append(lines, "", "## Projection Materialization")
		for _, stateValue := range []string{
			projectionMaterializationMaterialized,
			projectionMaterializationPartial,
			projectionMaterializationFailed,
		} {
			if count := materializationCounts[stateValue]; count > 0 {
				lines = append(lines, fmt.Sprintf("- %s: %d", stateValue, count))
			}
		}
	}
	if len(state.Workflows) > 0 {
		lines = append(lines, "", "## Workflows")
		for _, workflow := range state.Workflows {
			lines = append(lines, fmt.Sprintf("- [%s] %s (%s)", workflow.Status, workflow.Title, workflow.ID))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildProjectionMetricsView(state sessionState, docs []projectionRecord, resources []projectionRecord, skills []projectionRecord, events []controlPlaneEvent) projectionMetricsView {
	uniqueDenied := dedupeLines(state.DeniedPaths)
	return projectionMetricsView{
		ProjectionGeneration:     state.ProjectionGeneration,
		ProjectionBuildMS:        state.ProjectionBuildMS,
		ProjectionCounts:         countProjectionNamespaces(docs, resources, skills),
		FreshnessCounts:          countProjectionFreshness(docs, resources, skills),
		MaterializationCounts:    countProjectionMaterialization(docs, resources, skills),
		ControlPlaneEvents:       len(events),
		UniqueDeniedPaths:        len(uniqueDenied),
		CacheHitMetricsAvailable: false,
	}
}

func renderProjectionMetricsSummary(state sessionState, docs []projectionRecord, resources []projectionRecord, skills []projectionRecord, events []controlPlaneEvent) string {
	metrics := buildProjectionMetricsView(state, docs, resources, skills, events)
	lines := []string{
		"# Projection Metrics",
		"",
		fmt.Sprintf("- projection_generation: %d", metrics.ProjectionGeneration),
		fmt.Sprintf("- projection_build_ms: %d", metrics.ProjectionBuildMS),
		fmt.Sprintf("- projections.documents: %d", metrics.ProjectionCounts["documents"]),
		fmt.Sprintf("- projections.resources: %d", metrics.ProjectionCounts["resources"]),
		fmt.Sprintf("- projections.skills: %d", metrics.ProjectionCounts["skills"]),
		fmt.Sprintf("- control_plane_events: %d", metrics.ControlPlaneEvents),
		fmt.Sprintf("- unique_denied_paths: %d", metrics.UniqueDeniedPaths),
		fmt.Sprintf("- cache_hit_metrics_available: %t", metrics.CacheHitMetricsAvailable),
	}
	if len(metrics.FreshnessCounts) > 0 {
		lines = append(lines, "", "## Freshness")
		for _, freshness := range []string{
			projectionFreshnessLive,
			projectionFreshnessSnapshot,
			projectionFreshnessStale,
			projectionFreshnessUpdated,
			projectionFreshnessGenerated,
		} {
			if count := metrics.FreshnessCounts[freshness]; count > 0 {
				lines = append(lines, fmt.Sprintf("- %s: %d", freshness, count))
			}
		}
	}
	if len(metrics.MaterializationCounts) > 0 {
		lines = append(lines, "", "## Materialization")
		for _, stateValue := range []string{
			projectionMaterializationMaterialized,
			projectionMaterializationPartial,
			projectionMaterializationFailed,
		} {
			if count := metrics.MaterializationCounts[stateValue]; count > 0 {
				lines = append(lines, fmt.Sprintf("- %s: %d", stateValue, count))
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildDenialView(state sessionState) denialView {
	unique := dedupeLines(state.DeniedPaths)
	return denialView{
		ProjectionGeneration: state.ProjectionGeneration,
		UniqueDeniedPaths:    len(unique),
		ByNamespace:          countDenialsByNamespace(unique),
		SamplePaths:          append([]string(nil), unique...),
	}
}

func renderDenialSummary(state sessionState) string {
	view := buildDenialView(state)
	lines := []string{"# Denials", ""}
	if len(view.SamplePaths) == 0 {
		lines = append(lines, "- no denied paths")
		return strings.Join(lines, "\n") + "\n"
	}
	lines = append(lines, fmt.Sprintf("- projection_generation: %d", view.ProjectionGeneration))
	lines = append(lines, fmt.Sprintf("- unique_denied_paths: %d", view.UniqueDeniedPaths), "")
	for _, namespace := range []string{"reference", "resources", "skills", "memory", "external_or_unknown"} {
		if count := view.ByNamespace[namespace]; count > 0 {
			lines = append(lines, fmt.Sprintf("- %s: %d", namespace, count))
		}
	}
	lines = append(lines, "", "## Sample Paths")
	for _, pathValue := range view.SamplePaths {
		lines = append(lines, fmt.Sprintf("- %s", pathValue))
	}
	return strings.Join(lines, "\n") + "\n"
}

func controlPlaneEventViews(events []controlPlaneEvent, projectionGeneration int) []controlPlaneEventViewRecord {
	if len(events) == 0 {
		return []controlPlaneEventViewRecord{}
	}
	views := make([]controlPlaneEventViewRecord, 0, len(events))
	for _, event := range events {
		visibility := "visible"
		if projectionGeneration < event.VisibleFromGeneration {
			visibility = "pending"
		}
		views = append(views, controlPlaneEventViewRecord{
			Seq:                   event.Seq,
			Op:                    event.Op,
			Path:                  event.Path,
			SelectionScope:        event.SelectionScope,
			Result:                event.Result,
			Visibility:            visibility,
			VisibleAfter:          event.VisibleAfter,
			VisibleFromGeneration: event.VisibleFromGeneration,
			SelectedBefore:        event.SelectedBefore,
			SelectedAfter:         event.SelectedAfter,
			WinnerBefore:          event.WinnerBefore,
			WinnerAfter:           event.WinnerAfter,
			ReasonAfter:           event.ReasonAfter,
		})
	}
	return views
}

func renderControlPlaneEventSummary(events []controlPlaneEvent, projectionGeneration int) string {
	lines := []string{"# Skills Control-Plane Audit", ""}
	views := controlPlaneEventViews(events, projectionGeneration)
	if len(views) == 0 {
		lines = append(lines, "- no control-plane events")
		return strings.Join(lines, "\n") + "\n"
	}
	lines = append(lines, fmt.Sprintf("- projection_generation: %d", projectionGeneration))
	lines = append(lines, fmt.Sprintf("- events: %d", len(views)), "")
	for _, event := range views {
		line := fmt.Sprintf("- [%d] %s %s result=%s visibility=%s visible_from=%d", event.Seq, event.Op, event.Path, event.Result, event.Visibility, event.VisibleFromGeneration)
		if event.SelectionScope != "" {
			line += fmt.Sprintf(" scope=%s", event.SelectionScope)
		}
		if event.WinnerAfter != "" {
			line += fmt.Sprintf(" winner_after=%s", event.WinnerAfter)
		}
		if event.ReasonAfter != "" {
			line += fmt.Sprintf(" reason_after=%s", event.ReasonAfter)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderCuratedSummary(curated []curatedRecord) string {
	lines := []string{"# Curated Memory", ""}
	if len(curated) == 0 {
		lines = append(lines, "- no curated entries")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, entry := range curated {
		lines = append(lines, fmt.Sprintf("- [%s] %s (rev=%d source=%s)", entry.ID, entry.Title, entry.Revision, entry.Source))
		if entry.Summary != "" {
			lines = append(lines, fmt.Sprintf("  summary: %s", entry.Summary))
		}
		if len(entry.SourcePaths) > 0 {
			lines = append(lines, fmt.Sprintf("  source_paths: %s", strings.Join(entry.SourcePaths, ", ")))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func countProjectionFreshness(groups ...[]projectionRecord) map[string]int {
	counts := map[string]int{}
	for _, records := range groups {
		for _, record := range records {
			if record.Freshness == "" {
				continue
			}
			counts[record.Freshness]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func countProjectionMaterialization(groups ...[]projectionRecord) map[string]int {
	counts := map[string]int{}
	for _, records := range groups {
		for _, record := range records {
			state := normalizeProjectionMaterialization(record.Materialization, projectionMaterializationMaterialized).State
			if state == "" {
				continue
			}
			counts[state]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func countProjectionNamespaces(docs []projectionRecord, resources []projectionRecord, skills []projectionRecord) map[string]int {
	return map[string]int{
		"documents": len(docs),
		"resources": len(resources),
		"skills":    len(skills),
	}
}

func countDenialsByNamespace(paths []string) map[string]int {
	counts := map[string]int{
		"reference":           0,
		"resources":           0,
		"skills":              0,
		"memory":              0,
		"external_or_unknown": 0,
	}
	for _, pathValue := range paths {
		switch {
		case strings.HasPrefix(pathValue, referenceRoot+"/"):
			counts["reference"]++
		case strings.HasPrefix(pathValue, resourcesRoot+"/"):
			counts["resources"]++
		case strings.HasPrefix(pathValue, skillsRoot+"/"):
			counts["skills"]++
		case strings.HasPrefix(pathValue, memoryRoot+"/"):
			counts["memory"]++
		default:
			counts["external_or_unknown"]++
		}
	}
	return counts
}

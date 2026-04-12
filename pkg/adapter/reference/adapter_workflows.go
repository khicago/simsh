package reference

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

func summarizeTrace(trace contract.ExecutionTrace) []string {
	lines := make([]string, 0)
	for _, pathValue := range trace.ReadPaths {
		switch {
		case strings.HasPrefix(pathValue, referenceRoot+"/"):
			lines = append(lines, "read-ref:"+pathValue)
		case strings.HasPrefix(pathValue, resourcesRoot+"/"):
			lines = append(lines, "read-resource:"+pathValue)
		case strings.HasPrefix(pathValue, skillsRoot+"/"):
			lines = append(lines, "read-skill:"+pathValue)
		}
	}
	for _, group := range [][]string{trace.WrittenPaths, trace.EditedPaths, trace.AppendedPaths} {
		for _, pathValue := range group {
			lines = append(lines, "wrote:"+pathValue)
		}
	}
	for _, pathValue := range trace.DeniedPaths {
		lines = append(lines, "denied:"+pathValue)
	}
	for _, outcome := range trace.ExternalOutcomes {
		command := externalOutcomeObservationCommand(outcome)
		kind := strings.TrimSpace(string(outcome.OutcomeKind))
		if kind == "" {
			kind = "unknown"
		}
		lines = append(lines, fmt.Sprintf("external-outcome:%s:%s", kind, command))
	}
	return lines
}

func externalOutcomeObservationCommand(outcome contract.ExecutionTraceStep) string {
	command := strings.TrimSpace(outcome.Command)
	if command != "" {
		return command
	}
	resolvedPath := strings.TrimSpace(outcome.ResolvedPath)
	if resolvedPath != "" {
		return resolvedPath
	}
	for _, arg := range outcome.Argv {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			return arg
		}
	}
	return "unknown"
}

func collectPrefixedPaths(paths []string, prefix string) []string {
	out := make([]string, 0)
	for _, pathValue := range paths {
		if strings.HasPrefix(pathValue, prefix) {
			out = append(out, pathValue)
		}
	}
	return dedupeLines(out)
}

func collectOutputPaths(trace contract.ExecutionTrace) []string {
	paths := make([]string, 0, len(trace.WrittenPaths)+len(trace.EditedPaths)+len(trace.AppendedPaths))
	for _, group := range [][]string{trace.WrittenPaths, trace.EditedPaths, trace.AppendedPaths} {
		for _, pathValue := range group {
			if strings.HasPrefix(pathValue, taskOutputsRoot+"/") {
				paths = append(paths, pathValue)
			}
		}
	}
	return dedupeLines(paths)
}

func mergePaths(existing []string, newPaths []string) []string {
	return dedupeLines(append(existing, newPaths...))
}

func (a *Adapter) workflowViews(state sessionState) []workflowView {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.workflows) == 0 {
		return nil
	}
	views := make([]workflowView, 0, len(a.workflows))
	for _, workflow := range a.workflows {
		view := workflowView{
			ID:              workflow.ID,
			Title:           workflow.Title,
			Summary:         workflow.Summary,
			ResourcePaths:   append([]string(nil), workflow.ResourcePaths...),
			ExpectedOutputs: append([]string(nil), workflow.ExpectedOutputs...),
			Status:          "pending",
			StatusSource:    "trace",
		}
		evidence := make([]string, 0)
		switch {
		case allPathsPresent(workflow.ExpectedOutputs, state.WrittenOutputs):
			view.Status = "completed"
			evidence = append(evidence, state.WrittenOutputs...)
		case anyPathsPresent(workflow.ExpectedOutputs, state.DeniedPaths) || anyPathsPresent(workflow.ResourcePaths, state.DeniedPaths):
			view.Status = "blocked"
			evidence = append(evidence, state.DeniedPaths...)
		case anyPathsPresent(workflow.ResourcePaths, state.ReadResources) || anyPathsPresent(workflow.ExpectedOutputs, state.WrittenOutputs):
			view.Status = "in_progress"
			evidence = append(evidence, state.ReadResources...)
			evidence = append(evidence, state.WrittenOutputs...)
		}
		view.Evidence = filterWorkflowEvidence(evidence, workflow)
		if override, ok := a.workflowStatus[workflow.ID]; ok && override.Status != "" {
			view.Status = override.Status
			view.StatusSource = "control_plane"
			view.StatusReason = override.Reason
		}
		views = append(views, view)
	}
	return views
}

func allPathsPresent(expected []string, actual []string) bool {
	if len(expected) == 0 {
		return false
	}
	for _, pathValue := range expected {
		if !containsLine(actual, pathValue) {
			return false
		}
	}
	return true
}

func anyPathsPresent(expected []string, actual []string) bool {
	for _, pathValue := range expected {
		if containsLine(actual, pathValue) {
			return true
		}
	}
	return false
}

func filterWorkflowEvidence(evidence []string, workflow WorkflowSpec) []string {
	out := make([]string, 0, len(evidence))
	for _, item := range evidence {
		if item == "" {
			continue
		}
		if anyHasPrefix(item, workflow.ResourcePaths) || anyHasPrefix(item, workflow.ExpectedOutputs) {
			out = append(out, item)
		}
	}
	return dedupeLines(out)
}

func anyHasPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func containsLine(lines []string, target string) bool {
	for _, line := range lines {
		if line == target {
			return true
		}
	}
	return false
}

func renderWorkflowSummary(workflows []workflowView) string {
	lines := []string{"# Managed Workflows", ""}
	if len(workflows) == 0 {
		lines = append(lines, "- no workflows configured")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, workflow := range workflows {
		lines = append(lines, "- ["+workflow.Status+"] "+workflow.Title+" ("+workflow.ID+")")
		if workflow.StatusSource != "" {
			lines = append(lines, "  source: "+workflow.StatusSource)
		}
		if workflow.StatusReason != "" {
			lines = append(lines, "  reason: "+workflow.StatusReason)
		}
		if workflow.Summary != "" {
			lines = append(lines, "  summary: "+workflow.Summary)
		}
		if len(workflow.ResourcePaths) > 0 {
			lines = append(lines, "  resources: "+strings.Join(workflow.ResourcePaths, ", "))
		}
		if len(workflow.ExpectedOutputs) > 0 {
			lines = append(lines, "  outputs: "+strings.Join(workflow.ExpectedOutputs, ", "))
		}
		if len(workflow.Evidence) > 0 {
			lines = append(lines, "  evidence: "+strings.Join(workflow.Evidence, ", "))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func refreshProjectionEntry(existing projectionEntry, content string, meta ProjectionMetadata) projectionEntry {
	source := strings.TrimSpace(meta.Source)
	if source == "" {
		source = existing.Metadata.Source
	}
	if source == "" {
		source = "control_plane"
	}
	freshness := normalizeProjectionFreshness(meta.Freshness)
	if freshness == "" {
		freshness = projectionFreshnessLive
	}
	materialization := normalizeProjectionMaterialization(meta.Materialization, projectionMaterializationMaterialized)
	return projectionEntry{
		Content: content,
		Metadata: ProjectionMetadata{
			Source:          source,
			Freshness:       freshness,
			Materialization: materialization,
		},
	}
}

func refreshProjectionEntryInMap(entries map[string]projectionEntry, name string, content string, meta ProjectionMetadata) {
	existing, ok := entries[name]
	if !ok {
		return
	}
	entries[name] = refreshProjectionEntry(existing, content, meta)
}

func invalidateProjectionEntry(entries map[string]projectionEntry, name string) {
	entry, ok := entries[name]
	if !ok {
		return
	}
	materialization := normalizeProjectionMaterialization(entry.Metadata.Materialization, projectionMaterializationMaterialized)
	if materialization.State == projectionMaterializationMaterialized {
		materialization = ProjectionMaterialization{
			State:  projectionMaterializationPartial,
			Reason: "refresh_required",
		}
	}
	entry.Metadata = normalizeProjectionMetadata(
		ProjectionMetadata{
			Source:          entry.Metadata.Source,
			Freshness:       projectionFreshnessStale,
			Materialization: materialization,
		},
		entry.Metadata.Source,
		projectionFreshnessStale,
	)
	entries[name] = entry
}

func setProjectionMaterializationInMap(entries map[string]projectionEntry, name string, state string, reason string) {
	entry, ok := entries[name]
	if !ok {
		return
	}
	entry.Metadata.Materialization = normalizeProjectionMaterialization(
		ProjectionMaterialization{State: state, Reason: reason},
		projectionMaterializationMaterialized,
	)
	entries[name] = entry
}

func shouldMaterializeProjection(materialization ProjectionMaterialization) bool {
	return normalizeProjectionMaterialization(materialization, projectionMaterializationMaterialized).State != projectionMaterializationFailed
}

func newProjectionEntry(content string, meta ProjectionMetadata, defaultSource string, defaultFreshness string) projectionEntry {
	return projectionEntry{
		Content:  content,
		Metadata: normalizeProjectionMetadata(meta, defaultSource, defaultFreshness),
	}
}

func renderProjectionSummary(docs []projectionRecord, resources []projectionRecord, skills []projectionRecord) string {
	lines := []string{"# Projection Index", ""}
	if len(docs) == 0 && len(resources) == 0 && len(skills) == 0 {
		lines = append(lines, "- no projected items")
		return strings.Join(lines, "\n") + "\n"
	}
	if len(docs) > 0 {
		lines = append(lines, "## Documents")
		for _, record := range docs {
			line := "- " + record.Path + " [" + record.Source + "/" + record.Freshness + "]"
			if record.Materialization.State != "" && record.Materialization.State != projectionMaterializationMaterialized {
				line += " materialization=" + record.Materialization.State
				if record.Materialization.Reason != "" {
					line += " reason=" + record.Materialization.Reason
				}
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	if len(resources) > 0 {
		lines = append(lines, "## Resources")
		for _, record := range resources {
			line := "- " + record.Path + " [" + record.Source + "/" + record.Freshness + "]"
			if record.Materialization.State != "" && record.Materialization.State != projectionMaterializationMaterialized {
				line += " materialization=" + record.Materialization.State
				if record.Materialization.Reason != "" {
					line += " reason=" + record.Materialization.Reason
				}
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	if len(skills) > 0 {
		lines = append(lines, "## Skills")
		for _, record := range skills {
			line := "- " + record.Path + " [" + record.Source + "/" + record.Freshness + "]"
			if record.Materialization.State != "" && record.Materialization.State != projectionMaterializationMaterialized {
				line += " materialization=" + record.Materialization.State
				if record.Materialization.Reason != "" {
					line += " reason=" + record.Materialization.Reason
				}
			}
			if record.Eligibility != nil {
				line += " eligibility=" + record.Eligibility.State
			}
			if record.Precedence != nil {
				line += fmt.Sprintf(" precedence=%s:%d", record.Precedence.Tier, record.Precedence.Rank)
			}
			if record.Selection != nil {
				line += " selection=" + record.Selection.Mode + ":" + record.Selection.State
				if record.Selection.Scope != "" {
					line += " scope=" + record.Selection.Scope
				}
				if record.Selection.Reason != "" {
					line += " reason=" + record.Selection.Reason
				}
				if record.Selection.WinnerPath != "" {
					line += " winner=" + record.Selection.WinnerPath
				}
			}
			if record.Selected {
				line += " selected=true"
			}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

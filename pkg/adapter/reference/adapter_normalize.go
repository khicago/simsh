package reference

import (
	"path"
	"sort"
	"strings"
)

func normalizeDocumentName(name string) string {
	value := normalizeProjectionName(name, "empty.md")
	if value == "empty.md" {
		return value
	}
	if !strings.HasSuffix(value, ".md") {
		value += ".md"
	}
	return value
}

func normalizeResourceName(name string) string {
	return normalizeProjectionName(name, "resource.txt")
}

func normalizeSkillName(name string) string {
	value := normalizeProjectionName(name, "default/SKILL.md")
	if value == "default/SKILL.md" {
		return value
	}
	if path.Ext(value) == "" {
		value = path.Join(value, "SKILL.md")
	}
	return value
}

func normalizeProjectionName(name string, fallback string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		return fallback
	}
	cleaned := strings.TrimPrefix(path.Clean("/"+value), "/")
	if cleaned == "" || cleaned == "." {
		return fallback
	}
	return cleaned
}

func normalizeCuratedID(raw string) string {
	return normalizeProjectionName(raw, "")
}

func normalizeCuratedSourcePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "/") {
			value = "/" + value
		}
		value = path.Clean(value)
		if value == "/" {
			continue
		}
		out = append(out, value)
	}
	return dedupeLines(out)
}

func normalizeCuratedEntry(entry CuratedEntry, defaultSource string) (curatedEntry, bool) {
	id := normalizeCuratedID(entry.ID)
	if id == "" {
		return curatedEntry{}, false
	}
	sourcePaths := normalizeCuratedSourcePaths(entry.SourcePaths)
	if len(sourcePaths) == 0 {
		return curatedEntry{}, false
	}
	title := strings.TrimSpace(entry.Title)
	if title == "" {
		title = id
	}
	source := strings.TrimSpace(entry.Source)
	if source == "" {
		source = defaultSource
	}
	if source == "" {
		source = "control_plane"
	}
	return curatedEntry{
		ID:          id,
		Title:       title,
		Summary:     strings.TrimSpace(entry.Summary),
		Content:     strings.TrimSpace(entry.Content),
		Source:      source,
		SourcePaths: sourcePaths,
		Revision:    1,
	}, true
}

func normalizeProjectionMetadata(meta ProjectionMetadata, defaultSource string, defaultFreshness string) ProjectionMetadata {
	source := strings.TrimSpace(meta.Source)
	if source == "" {
		source = defaultSource
	}
	freshness := normalizeProjectionFreshness(meta.Freshness)
	if freshness == "" {
		freshness = normalizeProjectionFreshness(defaultFreshness)
	}
	return ProjectionMetadata{
		Source:          source,
		Freshness:       freshness,
		Materialization: normalizeProjectionMaterialization(meta.Materialization, projectionMaterializationMaterialized),
	}
}

func normalizeProjectionMaterialization(raw ProjectionMaterialization, defaultState string) ProjectionMaterialization {
	state := normalizeProjectionMaterializationState(raw.State)
	if state == "" {
		state = normalizeProjectionMaterializationState(defaultState)
	}
	reason := strings.TrimSpace(raw.Reason)
	if state == projectionMaterializationMaterialized {
		reason = ""
	}
	return ProjectionMaterialization{
		State:  state,
		Reason: reason,
	}
}

func normalizeProjectionMaterializationState(raw string) string {
	switch strings.TrimSpace(raw) {
	case projectionMaterializationMaterialized,
		projectionMaterializationPartial,
		projectionMaterializationFailed:
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func normalizeSkillMetadata(meta SkillMetadata, defaultSource string, defaultFreshness string) SkillMetadata {
	source := strings.TrimSpace(meta.Source)
	if source == "" {
		source = defaultSource
	}
	freshness := normalizeProjectionFreshness(meta.Freshness)
	if freshness == "" {
		freshness = normalizeProjectionFreshness(defaultFreshness)
	}
	eligibility := normalizeSkillEligibility(meta.Eligibility)
	if eligibility.State == "" {
		eligibility.State = skillEligibilityUnknown
	}
	precedence := normalizeSkillPrecedence(meta.Precedence)
	if precedence.Tier == "" {
		precedence.Tier = skillPrecedenceTierBundled
	}
	if precedence.Rank == 0 && strings.TrimSpace(meta.Precedence.Tier) == "" {
		precedence.Rank = 100
	}
	selectionScope := normalizeSelectionScope(meta.SelectionScope)
	selected := meta.Selected
	if eligibility.State != skillEligibilityEligible {
		selected = false
	}
	return SkillMetadata{
		Source:         source,
		Freshness:      freshness,
		Eligibility:    eligibility,
		Precedence:     precedence,
		SelectionScope: selectionScope,
		Selected:       selected,
	}
}

func normalizeSelectionScope(raw string) string {
	return normalizeProjectionName(raw, "")
}

func normalizeSkillEligibility(raw SkillEligibility) SkillEligibility {
	return SkillEligibility{
		State:  normalizeSkillEligibilityState(raw.State),
		Reason: strings.TrimSpace(raw.Reason),
	}
}

func normalizeSkillEligibilityState(raw string) string {
	switch strings.TrimSpace(raw) {
	case skillEligibilityEligible,
		skillEligibilityIneligible,
		skillEligibilityUnknown:
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func normalizeSkillPrecedence(raw SkillPrecedence) SkillPrecedence {
	rank := raw.Rank
	if rank < 0 {
		rank = 0
	}
	return SkillPrecedence{
		Tier: normalizeSkillPrecedenceTier(raw.Tier),
		Rank: rank,
	}
}

func normalizeSkillPrecedenceTier(raw string) string {
	switch strings.TrimSpace(raw) {
	case skillPrecedenceTierWorkspace,
		skillPrecedenceTierUser,
		skillPrecedenceTierBundled:
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func normalizeProjectionFreshness(raw string) string {
	switch strings.TrimSpace(raw) {
	case projectionFreshnessSnapshot,
		projectionFreshnessLive,
		projectionFreshnessStale,
		projectionFreshnessUpdated,
		projectionFreshnessGenerated:
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func normalizeSkillMetadataMap(raw map[string]SkillMetadata, normalizeName func(string) string) map[string]SkillMetadata {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]SkillMetadata, len(raw))
	for key, meta := range raw {
		if normalizeName == nil {
			continue
		}
		name := normalizeName(key)
		if strings.TrimSpace(name) == "" {
			continue
		}
		out[name] = meta
	}
	return out
}

func normalizeProjectionMetadataMap(raw map[string]ProjectionMetadata, normalizeName func(string) string) map[string]ProjectionMetadata {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]ProjectionMetadata, len(raw))
	for key, meta := range raw {
		if normalizeName == nil {
			continue
		}
		name := normalizeName(key)
		if strings.TrimSpace(name) == "" {
			continue
		}
		out[name] = meta
	}
	return out
}

func normalizeWorkflowSpec(spec WorkflowSpec) (WorkflowSpec, bool) {
	id := strings.TrimSpace(spec.ID)
	title := strings.TrimSpace(spec.Title)
	if id == "" || title == "" {
		return WorkflowSpec{}, false
	}
	spec.ID = id
	spec.Title = title
	spec.Summary = strings.TrimSpace(spec.Summary)
	spec.ResourcePaths = normalizeWorkflowPaths(spec.ResourcePaths, resourcesRoot)
	spec.ExpectedOutputs = normalizeWorkflowPaths(spec.ExpectedOutputs, "")
	return spec, true
}

func normalizeWorkflowStatus(raw string) string {
	switch strings.TrimSpace(raw) {
	case "pending", "in_progress", "completed", "blocked":
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func normalizeWorkflowPaths(paths []string, defaultRoot string) []string {
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if defaultRoot != "" && !strings.HasPrefix(value, "/") {
			value = path.Join(defaultRoot, value)
		}
		if !strings.HasPrefix(value, "/") {
			value = "/" + value
		}
		out = append(out, path.Clean(value))
	}
	return dedupeLines(out)
}

func dedupeLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	sort.Strings(out)
	return out
}

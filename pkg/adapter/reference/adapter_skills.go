package reference

import (
	"path"
	"sort"
	"strings"
)

func deriveSkillSelectionOutcomes(entries map[string]skillEntry) map[string]skillSelectionOutcome {
	outcomes := make(map[string]skillSelectionOutcome, len(entries))
	scopes := make(map[string][]string)
	for name, entry := range entries {
		meta := entry.Metadata
		if meta.SelectionScope == "" {
			selected := meta.Selected && meta.Eligibility.State == skillEligibilityEligible
			state := skillSelectionStateNotSelected
			reason := skillSelectionReasonExplicitNotSelected
			if selected {
				state = skillSelectionStateSelected
				reason = skillSelectionReasonExplicitSelected
			} else if meta.Eligibility.State == skillEligibilityIneligible {
				reason = skillSelectionReasonIneligible
			} else if meta.Eligibility.State == skillEligibilityUnknown {
				reason = skillSelectionReasonUnknownEligibility
			}
			outcomes[name] = skillSelectionOutcome{
				Selected: selected,
				Selection: &SkillSelection{
					State:  state,
					Mode:   skillSelectionModeExplicit,
					Reason: reason,
				},
			}
			continue
		}
		scopes[meta.SelectionScope] = append(scopes[meta.SelectionScope], name)
	}
	for scope, names := range scopes {
		sort.Slice(names, func(i, j int) bool {
			return compareSkillSelectionCandidate(names[i], entries[names[i]].Metadata, names[j], entries[names[j]].Metadata) < 0
		})
		winnerName, hasWinner := selectSkillWinner(names, entries)
		for _, name := range names {
			meta := entries[name].Metadata
			if meta.Eligibility.State != skillEligibilityEligible {
				outcomes[name] = skillSelectionOutcome{
					Selected: false,
					Selection: &SkillSelection{
						State:  skillSelectionStateNotSelected,
						Mode:   skillSelectionModeDerived,
						Scope:  scope,
						Reason: skillSelectionReasonIneligible,
					},
				}
				continue
			}
			if hasWinner && name == winnerName {
				outcomes[name] = skillSelectionOutcome{
					Selected: true,
					Selection: &SkillSelection{
						State:  skillSelectionStateSelected,
						Mode:   skillSelectionModeDerived,
						Scope:  scope,
						Reason: winnerSelectionReason(name, names, entries),
					},
				}
				continue
			}
			outcomes[name] = skillSelectionOutcome{
				Selected: false,
				Selection: &SkillSelection{
					State:      skillSelectionStateNotSelected,
					Mode:       skillSelectionModeDerived,
					Scope:      scope,
					Reason:     loserSelectionReason(name, winnerName, entries),
					WinnerPath: skillSelectionWinnerPath(winnerName),
				},
			}
		}
	}
	return outcomes
}

func selectSkillWinner(names []string, entries map[string]skillEntry) (string, bool) {
	for _, name := range names {
		if entries[name].Metadata.Eligibility.State == skillEligibilityEligible {
			return name, true
		}
	}
	return "", false
}

func compareSkillSelectionCandidate(leftName string, left SkillMetadata, rightName string, right SkillMetadata) int {
	leftEligible := left.Eligibility.State == skillEligibilityEligible
	rightEligible := right.Eligibility.State == skillEligibilityEligible
	switch {
	case leftEligible && !rightEligible:
		return -1
	case !leftEligible && rightEligible:
		return 1
	}
	leftTier := skillPrecedenceTierWeight(left.Precedence.Tier)
	rightTier := skillPrecedenceTierWeight(right.Precedence.Tier)
	switch {
	case leftTier < rightTier:
		return -1
	case leftTier > rightTier:
		return 1
	}
	switch {
	case left.Precedence.Rank < right.Precedence.Rank:
		return -1
	case left.Precedence.Rank > right.Precedence.Rank:
		return 1
	}
	return strings.Compare(leftName, rightName)
}

func skillPrecedenceTierWeight(tier string) int {
	switch tier {
	case skillPrecedenceTierWorkspace:
		return 0
	case skillPrecedenceTierUser:
		return 1
	case skillPrecedenceTierBundled:
		return 2
	default:
		return 3
	}
}

func winnerSelectionReason(winnerName string, orderedNames []string, entries map[string]skillEntry) string {
	for _, name := range orderedNames {
		if name == winnerName {
			continue
		}
		meta := entries[name].Metadata
		if meta.Eligibility.State != skillEligibilityEligible {
			continue
		}
		if sameSkillPrecedence(entries[winnerName].Metadata.Precedence, meta.Precedence) {
			return skillSelectionReasonTieBreakPathOrder
		}
		break
	}
	return skillSelectionReasonHighestPrecedence
}

func loserSelectionReason(name string, winnerName string, entries map[string]skillEntry) string {
	if winnerName == "" {
		return skillSelectionReasonNoEligibleWinner
	}
	if sameSkillPrecedence(entries[name].Metadata.Precedence, entries[winnerName].Metadata.Precedence) {
		return skillSelectionReasonTieBreakPathOrder
	}
	return skillSelectionReasonHigherPrecedence
}

func sameSkillPrecedence(left SkillPrecedence, right SkillPrecedence) bool {
	return left.Tier == right.Tier && left.Rank == right.Rank
}

func skillSelectionWinnerPath(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return path.Join(skillsRoot, name)
}

func newSkillEntry(content string, meta SkillMetadata, defaultSource string, defaultFreshness string) skillEntry {
	return skillEntry{
		Content:  content,
		Metadata: normalizeSkillMetadata(meta, defaultSource, defaultFreshness),
	}
}

func cloneSkillEntries(entries map[string]skillEntry) map[string]skillEntry {
	if len(entries) == 0 {
		return map[string]skillEntry{}
	}
	cloned := make(map[string]skillEntry, len(entries))
	for name, entry := range entries {
		cloned[name] = entry
	}
	return cloned
}

func (a *Adapter) appendSkillControlPlaneEventLocked(kind string, normalizedName string, beforeSkills map[string]skillEntry, afterSkills map[string]skillEntry) {
	beforeOutcomes := deriveSkillSelectionOutcomes(beforeSkills)
	afterOutcomes := deriveSkillSelectionOutcomes(afterSkills)
	a.controlPlaneSeq++
	a.controlPlaneEvents = append(a.controlPlaneEvents, controlPlaneEvent{
		Seq:                   a.controlPlaneSeq,
		Op:                    kind,
		Path:                  skillSelectionWinnerPath(normalizedName),
		SelectionScope:        skillSelectionScopeForEvent(normalizedName, beforeSkills, afterSkills),
		Result:                "applied",
		VisibleAfter:          controlPlaneVisibilityNextProjection,
		VisibleFromGeneration: a.nextProjectionGenerationLocked() + len(a.controlPlaneEvents),
		SelectedBefore:        skillSelectionSelectedBefore(normalizedName, beforeOutcomes),
		SelectedAfter:         skillSelectionSelectedAfter(normalizedName, afterOutcomes),
		WinnerBefore:          skillSelectionWinnerBefore(normalizedName, beforeSkills, afterSkills, beforeOutcomes),
		WinnerAfter:           skillSelectionWinnerAfter(normalizedName, beforeSkills, afterSkills, afterOutcomes),
		ReasonAfter:           skillSelectionReasonAfter(normalizedName, afterOutcomes),
	})
}

func skillSelectionScopeForEvent(name string, beforeSkills map[string]skillEntry, afterSkills map[string]skillEntry) string {
	if entry, ok := afterSkills[name]; ok {
		return entry.Metadata.SelectionScope
	}
	if entry, ok := beforeSkills[name]; ok {
		return entry.Metadata.SelectionScope
	}
	return ""
}

func skillSelectionSelectedBefore(name string, outcomes map[string]skillSelectionOutcome) bool {
	if outcome, ok := outcomes[name]; ok {
		return outcome.Selected
	}
	return false
}

func skillSelectionSelectedAfter(name string, outcomes map[string]skillSelectionOutcome) bool {
	if outcome, ok := outcomes[name]; ok {
		return outcome.Selected
	}
	return false
}

func skillSelectionWinnerBefore(name string, beforeSkills map[string]skillEntry, afterSkills map[string]skillEntry, outcomes map[string]skillSelectionOutcome) string {
	return skillSelectionWinnerForScope(skillSelectionScopeForEvent(name, beforeSkills, afterSkills), beforeSkills, outcomes)
}

func skillSelectionWinnerAfter(name string, beforeSkills map[string]skillEntry, afterSkills map[string]skillEntry, outcomes map[string]skillSelectionOutcome) string {
	return skillSelectionWinnerForScope(skillSelectionScopeForEvent(name, beforeSkills, afterSkills), afterSkills, outcomes)
}

func skillSelectionWinnerForScope(scope string, skills map[string]skillEntry, outcomes map[string]skillSelectionOutcome) string {
	if strings.TrimSpace(scope) == "" {
		return ""
	}
	for name, entry := range skills {
		if entry.Metadata.SelectionScope != scope {
			continue
		}
		if outcome, ok := outcomes[name]; ok && outcome.Selected {
			return skillSelectionWinnerPath(name)
		}
	}
	return ""
}

func skillSelectionReasonAfter(name string, outcomes map[string]skillSelectionOutcome) string {
	if outcome, ok := outcomes[name]; ok && outcome.Selection != nil {
		return outcome.Selection.Reason
	}
	return ""
}

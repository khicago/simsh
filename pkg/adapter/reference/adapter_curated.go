package reference

import (
	"sort"
	"strings"
)

func upsertCuratedEntry(entries map[string]curatedEntry, entry curatedEntry) {
	if entry.ID == "" {
		return
	}
	existing, ok := entries[entry.ID]
	if ok {
		entry.Revision = existing.Revision + 1
	}
	if entry.Revision < 1 {
		entry.Revision = 1
	}
	entries[entry.ID] = entry
}

func mergeCuratedRecords(existing []curatedRecord, current []curatedRecord) []curatedRecord {
	if len(existing) == 0 && len(current) == 0 {
		return nil
	}
	merged := make(map[string]curatedRecord, len(existing)+len(current))
	for _, entry := range existing {
		if strings.TrimSpace(entry.ID) == "" {
			continue
		}
		merged[entry.ID] = entry
	}
	for _, entry := range current {
		if strings.TrimSpace(entry.ID) == "" {
			continue
		}
		merged[entry.ID] = entry
	}
	out := make([]curatedRecord, 0, len(merged))
	for _, entry := range merged {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

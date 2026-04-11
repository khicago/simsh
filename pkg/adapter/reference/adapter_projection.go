package reference

import (
	"encoding/json"
	"path"
	"sort"
	"time"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

func (a *Adapter) buildProjection(state sessionState) (contract.AdapterProjection, error) {
	startedAt := time.Now()
	projectionGeneration, controlPlaneEvents := a.beginProjectionBuild()
	state.Curated = mergeCuratedRecords(state.Curated, a.curatedRecords())
	if state.Workflows == nil {
		state.Workflows = a.workflowViews(state)
	}
	state.ProjectionGeneration = projectionGeneration
	state.ControlPlaneEvents = len(controlPlaneEvents)
	if len(controlPlaneEvents) > 0 {
		state.LastControlPlaneKind = controlPlaneEvents[len(controlPlaneEvents)-1].Op
	}
	docs := a.documentRecords()
	resources := a.resourceRecords()
	skills := a.skillRecords()
	curated := a.curatedRecords()
	state.ProjectionBuildMS = time.Since(startedAt).Milliseconds()
	raw, err := json.Marshal(state)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	knowledgeMount, err := a.knowledgeMount(docs)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	mounts := []contract.VirtualMount{knowledgeMount}
	resourceMount, err := a.resourceMount(resources)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	if resourceMount != nil {
		mounts = append(mounts, resourceMount)
	}
	skillMount, err := a.skillMount(skills)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	if skillMount != nil {
		mounts = append(mounts, skillMount)
	}
	memoryMount, err := mount.NewStaticMount(memoryRoot, "memory", memoryViewFiles(state, raw, docs, resources, skills, curated, controlPlaneEvents))
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	return contract.AdapterProjection{
		VirtualMounts: mounts,
		Memory: contract.MemoryProjection{
			Mount:     memoryMount,
			Freshness: state.Freshness,
		},
		OpaqueState: raw,
	}, nil
}

func (a *Adapter) knowledgeMount(records []projectionRecord) (contract.VirtualMount, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.projectionError != nil {
		return nil, a.projectionError
	}
	files := make(map[string]string, len(a.documents)+1)
	if len(a.documents) == 0 {
		files[path.Join(referenceRoot, "empty.md")] = "# Empty\n"
		records = []projectionRecord{{
			Path:      path.Join(referenceRoot, "empty.md"),
			Name:      "empty.md",
			Namespace: "reference",
			Kind:      "document",
			Source:    "adapter_default",
			Freshness: "generated",
			Materialization: ProjectionMaterialization{
				State: projectionMaterializationMaterialized,
			},
		}}
	}
	for name, entry := range a.documents {
		if !shouldMaterializeProjection(entry.Metadata.Materialization) {
			continue
		}
		files[path.Join(referenceRoot, name)] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files[path.Join(referenceRoot, "_index.json")] = string(raw)
	}
	return mount.NewStaticMount(referenceRoot, "reference", files)
}

func (a *Adapter) resourceMount(records []projectionRecord) (contract.VirtualMount, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.projectionError != nil {
		return nil, a.projectionError
	}
	if len(a.resources) == 0 {
		return nil, nil
	}
	files := make(map[string]string, len(a.resources)+1)
	for name, entry := range a.resources {
		if !shouldMaterializeProjection(entry.Metadata.Materialization) {
			continue
		}
		files[path.Join(resourcesRoot, name)] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files[path.Join(resourcesRoot, "_index.json")] = string(raw)
	}
	return mount.NewStaticMount(resourcesRoot, "resource", files)
}

func (a *Adapter) skillMount(records []projectionRecord) (contract.VirtualMount, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.projectionError != nil {
		return nil, a.projectionError
	}
	if len(a.skills) == 0 {
		return nil, nil
	}
	files := make(map[string]string, len(a.skills)+1)
	for name, entry := range a.skills {
		files[path.Join(skillsRoot, name)] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files[path.Join(skillsRoot, "_index.json")] = string(raw)
	}
	return mount.NewStaticMount(skillsRoot, "skill", files)
}

func (a *Adapter) documentRecords() []projectionRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return projectionRecordsFromMap(a.documents, referenceRoot, "reference", "document")
}

func (a *Adapter) resourceRecords() []projectionRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return projectionRecordsFromMap(a.resources, resourcesRoot, "resources", "resource")
}

func (a *Adapter) skillRecords() []projectionRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.skills) == 0 {
		return nil
	}
	outcomes := deriveSkillSelectionOutcomes(a.skills)
	out := make([]projectionRecord, 0, len(a.skills))
	for name, entry := range a.skills {
		eligibility := entry.Metadata.Eligibility
		precedence := entry.Metadata.Precedence
		outcome := outcomes[name]
		out = append(out, projectionRecord{
			Path:      path.Join(skillsRoot, name),
			Name:      name,
			Namespace: "skills",
			Kind:      "skill",
			Source:    entry.Metadata.Source,
			Freshness: entry.Metadata.Freshness,
			Materialization: ProjectionMaterialization{
				State: projectionMaterializationMaterialized,
			},
			Eligibility: &eligibility,
			Precedence:  &precedence,
			Selection:   outcome.Selection,
			Selected:    outcome.Selected,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func (a *Adapter) curatedRecords() []curatedRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.curated) == 0 {
		return nil
	}
	out := make([]curatedRecord, 0, len(a.curated))
	for _, entry := range a.curated {
		out = append(out, curatedRecord{
			ID:          entry.ID,
			Title:       entry.Title,
			Summary:     entry.Summary,
			Content:     entry.Content,
			Source:      entry.Source,
			SourcePaths: append([]string(nil), entry.SourcePaths...),
			Revision:    entry.Revision,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func projectionRecordsFromMap(entries map[string]projectionEntry, root string, namespace string, kind string) []projectionRecord {
	if len(entries) == 0 {
		return nil
	}
	out := make([]projectionRecord, 0, len(entries))
	for name, entry := range entries {
		out = append(out, projectionRecord{
			Path:            path.Join(root, name),
			Name:            name,
			Namespace:       namespace,
			Kind:            kind,
			Source:          entry.Metadata.Source,
			Freshness:       entry.Metadata.Freshness,
			Materialization: normalizeProjectionMaterialization(entry.Metadata.Materialization, projectionMaterializationMaterialized),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

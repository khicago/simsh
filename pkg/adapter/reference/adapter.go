package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

type Options struct {
	Documents        map[string]string
	DocumentMetadata map[string]ProjectionMetadata
	Resources        map[string]string
	ResourceMetadata map[string]ProjectionMetadata
	Skills           map[string]string
	SkillMetadata    map[string]SkillMetadata
	Curated          []CuratedEntry
	Workflows        []WorkflowSpec
}

type ProjectionMetadata struct {
	Source          string                    `json:"source,omitempty"`
	Freshness       string                    `json:"freshness,omitempty"`
	Materialization ProjectionMaterialization `json:"materialization"`
}

type ProjectionMaterialization struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type SkillMetadata struct {
	Source         string           `json:"source,omitempty"`
	Freshness      string           `json:"freshness,omitempty"`
	Eligibility    SkillEligibility `json:"eligibility"`
	Precedence     SkillPrecedence  `json:"precedence"`
	SelectionScope string           `json:"selection_scope,omitempty"`
	Selected       bool             `json:"selected,omitempty"`
}

type SkillEligibility struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type SkillPrecedence struct {
	Tier string `json:"tier"`
	Rank int    `json:"rank"`
}

type SkillSelection struct {
	State      string `json:"state"`
	Mode       string `json:"mode,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Reason     string `json:"reason,omitempty"`
	WinnerPath string `json:"winner_path,omitempty"`
}

const (
	projectionFreshnessSnapshot  = "snapshot"
	projectionFreshnessLive      = "live"
	projectionFreshnessStale     = "stale"
	projectionFreshnessUpdated   = "updated"
	projectionFreshnessGenerated = "generated"

	projectionMaterializationMaterialized = "materialized"
	projectionMaterializationPartial      = "partial"
	projectionMaterializationFailed       = "failed"

	skillEligibilityEligible   = "eligible"
	skillEligibilityIneligible = "ineligible"
	skillEligibilityUnknown    = "unknown"

	skillPrecedenceTierWorkspace = "workspace"
	skillPrecedenceTierUser      = "user"
	skillPrecedenceTierBundled   = "bundled"

	skillSelectionStateSelected    = "selected"
	skillSelectionStateNotSelected = "not_selected"

	skillSelectionModeDerived  = "derived"
	skillSelectionModeExplicit = "explicit"

	skillSelectionReasonExplicitSelected    = "explicit_selected"
	skillSelectionReasonExplicitNotSelected = "explicit_not_selected"
	skillSelectionReasonUnknownEligibility  = "unknown_eligibility"
	skillSelectionReasonIneligible          = "ineligible"
	skillSelectionReasonHighestPrecedence   = "highest_precedence"
	skillSelectionReasonHigherPrecedence    = "higher_precedence_selected"
	skillSelectionReasonTieBreakPathOrder   = "tie_breaker_path_order"
	skillSelectionReasonNoEligibleWinner    = "no_eligible_winner"

	controlPlaneEventKindSkillAdded   = "skill_added"
	controlPlaneEventKindSkillUpdated = "skill_updated"
	controlPlaneEventKindSkillRemoved = "skill_removed"

	controlPlaneVisibilityNextProjection = "next_projection_rebuild"
)

type WorkflowSpec struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary,omitempty"`
	ResourcePaths   []string `json:"resource_paths,omitempty"`
	ExpectedOutputs []string `json:"expected_outputs,omitempty"`
}

type CuratedEntry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Content     string   `json:"content,omitempty"`
	Source      string   `json:"source,omitempty"`
	SourcePaths []string `json:"source_paths,omitempty"`
}

type Adapter struct {
	mu                 sync.RWMutex
	documents          map[string]projectionEntry
	resources          map[string]projectionEntry
	skills             map[string]skillEntry
	curated            map[string]curatedEntry
	workflows          []WorkflowSpec
	workflowStatus     map[string]workflowStatusOverride
	controlPlaneEvents []controlPlaneEvent
	projectionSeq      int
	controlPlaneSeq    int
	projectionError    error
}

type projectionEntry struct {
	Content  string             `json:"content"`
	Metadata ProjectionMetadata `json:"metadata"`
}

type projectionRecord struct {
	Path            string                    `json:"path"`
	Name            string                    `json:"name"`
	Namespace       string                    `json:"namespace"`
	Kind            string                    `json:"kind"`
	Source          string                    `json:"source"`
	Freshness       string                    `json:"freshness"`
	Materialization ProjectionMaterialization `json:"materialization"`
	Eligibility     *SkillEligibility         `json:"eligibility,omitempty"`
	Precedence      *SkillPrecedence          `json:"precedence,omitempty"`
	Selection       *SkillSelection           `json:"selection,omitempty"`
	Selected        bool                      `json:"selected,omitempty"`
}

type projectionView struct {
	Documents []projectionRecord `json:"documents,omitempty"`
	Resources []projectionRecord `json:"resources,omitempty"`
	Skills    []projectionRecord `json:"skills,omitempty"`
}

type sessionState struct {
	Observations         []string        `json:"observations"`
	Freshness            string          `json:"freshness"`
	ReadRefs             []string        `json:"read_refs,omitempty"`
	ReadResources        []string        `json:"read_resources,omitempty"`
	ReadSkills           []string        `json:"read_skills,omitempty"`
	WrittenOutputs       []string        `json:"written_outputs,omitempty"`
	DeniedPaths          []string        `json:"denied_paths,omitempty"`
	Curated              []curatedRecord `json:"curated,omitempty"`
	Workflows            []workflowView  `json:"workflows,omitempty"`
	ProjectionGeneration int             `json:"projection_generation,omitempty"`
	ControlPlaneEvents   int             `json:"control_plane_events,omitempty"`
	LastControlPlaneKind string          `json:"last_control_plane_kind,omitempty"`
}

type workflowView struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary,omitempty"`
	ResourcePaths   []string `json:"resource_paths,omitempty"`
	ExpectedOutputs []string `json:"expected_outputs,omitempty"`
	Status          string   `json:"status"`
	StatusSource    string   `json:"status_source,omitempty"`
	StatusReason    string   `json:"status_reason,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
}

type workflowStatusOverride struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type skillEntry struct {
	Content  string        `json:"content"`
	Metadata SkillMetadata `json:"metadata"`
}

type skillSelectionOutcome struct {
	Selected  bool
	Selection *SkillSelection
}

type controlPlaneEvent struct {
	Seq                   int    `json:"seq"`
	Op                    string `json:"op"`
	Path                  string `json:"path"`
	SelectionScope        string `json:"selection_scope,omitempty"`
	Result                string `json:"result"`
	VisibleAfter          string `json:"visible_after"`
	VisibleFromGeneration int    `json:"visible_from_generation"`
	SelectedBefore        bool   `json:"selected_before"`
	SelectedAfter         bool   `json:"selected_after"`
	WinnerBefore          string `json:"winner_before,omitempty"`
	WinnerAfter           string `json:"winner_after,omitempty"`
	ReasonAfter           string `json:"reason_after,omitempty"`
}

type controlPlaneEventViewRecord struct {
	Seq                   int    `json:"seq"`
	Op                    string `json:"op"`
	Path                  string `json:"path"`
	SelectionScope        string `json:"selection_scope,omitempty"`
	Result                string `json:"result"`
	Visibility            string `json:"visibility"`
	VisibleAfter          string `json:"visible_after"`
	VisibleFromGeneration int    `json:"visible_from_generation"`
	SelectedBefore        bool   `json:"selected_before"`
	SelectedAfter         bool   `json:"selected_after"`
	WinnerBefore          string `json:"winner_before,omitempty"`
	WinnerAfter           string `json:"winner_after,omitempty"`
	ReasonAfter           string `json:"reason_after,omitempty"`
}

type curatedEntry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary,omitempty"`
	Content     string   `json:"content,omitempty"`
	Source      string   `json:"source"`
	SourcePaths []string `json:"source_paths"`
	Revision    int      `json:"revision"`
}

type curatedRecord struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary,omitempty"`
	Content     string   `json:"content,omitempty"`
	Source      string   `json:"source"`
	SourcePaths []string `json:"source_paths"`
	Revision    int      `json:"revision"`
}

func New(opts Options) *Adapter {
	documentMetadata := normalizeProjectionMetadataMap(opts.DocumentMetadata, normalizeDocumentName)
	documents := map[string]projectionEntry{}
	for key, value := range opts.Documents {
		name := normalizeDocumentName(key)
		documents[name] = projectionEntry{
			Content:  value,
			Metadata: normalizeProjectionMetadata(documentMetadata[name], "reference_bundle", "snapshot"),
		}
	}
	resourceMetadata := normalizeProjectionMetadataMap(opts.ResourceMetadata, normalizeResourceName)
	resources := map[string]projectionEntry{}
	for key, value := range opts.Resources {
		name := normalizeResourceName(key)
		resources[name] = projectionEntry{
			Content:  value,
			Metadata: normalizeProjectionMetadata(resourceMetadata[name], "resource_bundle", "snapshot"),
		}
	}
	skillMetadata := normalizeSkillMetadataMap(opts.SkillMetadata, normalizeSkillName)
	skills := map[string]skillEntry{}
	for key, value := range opts.Skills {
		name := normalizeSkillName(key)
		skills[name] = newSkillEntry(value, skillMetadata[name], "skill_bundle", projectionFreshnessSnapshot)
	}
	curated := map[string]curatedEntry{}
	for _, value := range opts.Curated {
		entry, ok := normalizeCuratedEntry(value, "curation_seed")
		if !ok {
			continue
		}
		upsertCuratedEntry(curated, entry)
	}
	workflows := make([]WorkflowSpec, 0, len(opts.Workflows))
	for _, workflow := range opts.Workflows {
		if normalized, ok := normalizeWorkflowSpec(workflow); ok {
			workflows = append(workflows, normalized)
		}
	}
	return &Adapter{
		documents:      documents,
		resources:      resources,
		skills:         skills,
		curated:        curated,
		workflows:      workflows,
		workflowStatus: map[string]workflowStatusOverride{},
	}
}

func (a *Adapter) AdapterID() string { return "reference" }

func (a *Adapter) CreateSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	return a.buildProjection(a.stateFromSession(session, "created"))
}

func (a *Adapter) ResumeSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	return a.buildProjection(a.stateFromSession(session, "resumed"))
}

func (a *Adapter) ObserveExecution(ctx context.Context, session contract.Session, result contract.ExecutionResult) (contract.AdapterProjection, error) {
	_ = ctx
	state := a.stateFromSession(session, "observed")
	state.Observations = dedupeLines(append(state.Observations, summarizeTrace(result.Trace)...))
	state.ReadRefs = mergePaths(state.ReadRefs, collectPrefixedPaths(result.Trace.ReadPaths, "/knowledge_base/reference/"))
	state.ReadResources = mergePaths(state.ReadResources, collectPrefixedPaths(result.Trace.ReadPaths, "/resources/"))
	state.ReadSkills = mergePaths(state.ReadSkills, collectPrefixedPaths(result.Trace.ReadPaths, "/skills/"))
	state.WrittenOutputs = mergePaths(state.WrittenOutputs, collectOutputPaths(result.Trace))
	state.DeniedPaths = mergePaths(state.DeniedPaths, result.Trace.DeniedPaths)
	state.Workflows = a.workflowViews(state)
	return a.buildProjection(state)
}

func (a *Adapter) CheckpointSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	return a.buildProjection(a.stateFromSession(session, "checkpointed"))
}

func (a *Adapter) CloseSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	return a.buildProjection(a.stateFromSession(session, "closed"))
}

func (a *Adapter) UpdateDocument(name string, content string) {
	a.UpsertDocument(name, content, ProjectionMetadata{Source: "control_plane", Freshness: "updated"})
}

func (a *Adapter) UpsertDocument(name string, content string, meta ProjectionMetadata) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.documents[normalizeDocumentName(name)] = newProjectionEntry(content, meta, "control_plane", projectionFreshnessUpdated)
}

func (a *Adapter) UpdateResource(name string, content string) {
	a.UpsertResource(name, content, ProjectionMetadata{Source: "control_plane", Freshness: "updated"})
}

func (a *Adapter) UpsertResource(name string, content string, meta ProjectionMetadata) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resources[normalizeResourceName(name)] = newProjectionEntry(content, meta, "control_plane", projectionFreshnessUpdated)
}

func (a *Adapter) UpdateSkill(name string, content string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	normalizedName := normalizeSkillName(name)
	if a.skills == nil {
		return
	}
	beforeSkills := cloneSkillEntries(a.skills)
	entry, ok := beforeSkills[normalizedName]
	if !ok {
		return
	}
	entry.Content = content
	entry.Metadata.Source = "control_plane"
	entry.Metadata.Freshness = projectionFreshnessUpdated
	a.skills[normalizedName] = entry
	a.appendSkillControlPlaneEventLocked(controlPlaneEventKindSkillUpdated, normalizedName, beforeSkills, cloneSkillEntries(a.skills))
}

func (a *Adapter) UpsertSkill(name string, content string, meta SkillMetadata) {
	a.mu.Lock()
	defer a.mu.Unlock()
	beforeSkills := cloneSkillEntries(a.skills)
	if a.skills == nil {
		a.skills = map[string]skillEntry{}
	}
	normalized := normalizeSkillName(name)
	a.skills[normalized] = newSkillEntry(content, meta, "control_plane", projectionFreshnessUpdated)
	kind := controlPlaneEventKindSkillAdded
	if _, existed := beforeSkills[normalized]; existed {
		kind = controlPlaneEventKindSkillUpdated
	}
	a.appendSkillControlPlaneEventLocked(kind, normalized, beforeSkills, cloneSkillEntries(a.skills))
}

func (a *Adapter) RemoveSkill(name string) {
	if normalized := normalizeSkillName(name); normalized != "" {
		a.mu.Lock()
		defer a.mu.Unlock()
		if len(a.skills) == 0 {
			return
		}
		beforeSkills := cloneSkillEntries(a.skills)
		if _, ok := beforeSkills[normalized]; !ok {
			return
		}
		delete(a.skills, normalized)
		a.appendSkillControlPlaneEventLocked(controlPlaneEventKindSkillRemoved, normalized, beforeSkills, cloneSkillEntries(a.skills))
	}
}

func (a *Adapter) RefreshDocument(name string, content string, meta ProjectionMetadata) {
	a.mu.Lock()
	defer a.mu.Unlock()
	refreshProjectionEntryInMap(a.documents, normalizeDocumentName(name), content, meta)
}

func (a *Adapter) RefreshResource(name string, content string, meta ProjectionMetadata) {
	a.mu.Lock()
	defer a.mu.Unlock()
	refreshProjectionEntryInMap(a.resources, normalizeResourceName(name), content, meta)
}

func (a *Adapter) InvalidateDocument(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	invalidateProjectionEntry(a.documents, normalizeDocumentName(name))
}

func (a *Adapter) InvalidateResource(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	invalidateProjectionEntry(a.resources, normalizeResourceName(name))
}

func (a *Adapter) SetDocumentMaterialization(name string, state string, reason string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	setProjectionMaterializationInMap(a.documents, normalizeDocumentName(name), state, reason)
}

func (a *Adapter) SetResourceMaterialization(name string, state string, reason string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	setProjectionMaterializationInMap(a.resources, normalizeResourceName(name), state, reason)
}

func (a *Adapter) UpsertWorkflow(spec WorkflowSpec) {
	normalized, ok := normalizeWorkflowSpec(spec)
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for idx, workflow := range a.workflows {
		if workflow.ID == normalized.ID {
			a.workflows[idx] = normalized
			return
		}
	}
	a.workflows = append(a.workflows, normalized)
}

func (a *Adapter) SetWorkflowStatus(id string, status string, reason string) {
	id = strings.TrimSpace(id)
	status = normalizeWorkflowStatus(status)
	if id == "" || status == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.workflowStatus == nil {
		a.workflowStatus = map[string]workflowStatusOverride{}
	}
	a.workflowStatus[id] = workflowStatusOverride{
		Status: status,
		Reason: strings.TrimSpace(reason),
	}
}

func (a *Adapter) ClearWorkflowStatus(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.workflowStatus, id)
}

func (a *Adapter) SetProjectionError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectionError = err
}

func (a *Adapter) nextProjectionGenerationLocked() int {
	return a.projectionSeq + 1
}

func (a *Adapter) beginProjectionBuild() (int, []controlPlaneEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectionSeq++
	events := append([]controlPlaneEvent(nil), a.controlPlaneEvents...)
	return a.projectionSeq, events
}

func (a *Adapter) UpsertCuratedEntry(entry CuratedEntry) {
	normalized, ok := normalizeCuratedEntry(entry, "control_plane")
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.curated == nil {
		a.curated = map[string]curatedEntry{}
	}
	upsertCuratedEntry(a.curated, normalized)
}

func (a *Adapter) RemoveCuratedEntry(id string) {
	id = normalizeCuratedID(id)
	if id == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.curated, id)
}

func (a *Adapter) buildProjection(state sessionState) (contract.AdapterProjection, error) {
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
	raw, err := json.Marshal(state)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	docs := a.documentRecords()
	resources := a.resourceRecords()
	skills := a.skillRecords()
	curated := a.curatedRecords()
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
	memoryMount, err := mount.NewStaticMount("/memory", "memory", memoryViewFiles(state, raw, docs, resources, skills, curated, controlPlaneEvents))
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
		files["/knowledge_base/reference/empty.md"] = "# Empty\n"
		records = []projectionRecord{{
			Path:      "/knowledge_base/reference/empty.md",
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
		files["/knowledge_base/reference/"+name] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files["/knowledge_base/reference/_index.json"] = string(raw)
	}
	return mount.NewStaticMount("/knowledge_base/reference", "reference", files)
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
		files["/resources/"+name] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files["/resources/_index.json"] = string(raw)
	}
	return mount.NewStaticMount("/resources", "resource", files)
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
		files["/skills/"+name] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files["/skills/_index.json"] = string(raw)
	}
	return mount.NewStaticMount("/skills", "skill", files)
}

func (a *Adapter) documentRecords() []projectionRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return projectionRecordsFromMap(a.documents, "/knowledge_base/reference", "reference", "document")
}

func (a *Adapter) resourceRecords() []projectionRecord {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return projectionRecordsFromMap(a.resources, "/resources", "resources", "resource")
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
			Path:      path.Join("/skills", name),
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

func (a *Adapter) stateFromSession(session contract.Session, freshness string) sessionState {
	state := sessionState{
		Freshness: freshness,
		Curated:   a.curatedRecords(),
		Workflows: a.workflowViews(sessionState{}),
	}
	if raw := session.State.Opaque[a.AdapterID()]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &state)
		state.Freshness = freshness
		state.Curated = mergeCuratedRecords(state.Curated, a.curatedRecords())
		state.Workflows = a.workflowViews(state)
	}
	return state
}

func summarizeTrace(trace contract.ExecutionTrace) []string {
	lines := make([]string, 0)
	for _, pathValue := range trace.ReadPaths {
		switch {
		case strings.HasPrefix(pathValue, "/knowledge_base/reference/"):
			lines = append(lines, fmt.Sprintf("read-ref:%s", pathValue))
		case strings.HasPrefix(pathValue, "/resources/"):
			lines = append(lines, fmt.Sprintf("read-resource:%s", pathValue))
		case strings.HasPrefix(pathValue, "/skills/"):
			lines = append(lines, fmt.Sprintf("read-skill:%s", pathValue))
		}
	}
	for _, group := range [][]string{trace.WrittenPaths, trace.EditedPaths, trace.AppendedPaths} {
		for _, pathValue := range group {
			lines = append(lines, fmt.Sprintf("wrote:%s", pathValue))
		}
	}
	for _, pathValue := range trace.DeniedPaths {
		lines = append(lines, fmt.Sprintf("denied:%s", pathValue))
	}
	return lines
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
	return path.Join("/skills", name)
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

func newProjectionEntry(content string, meta ProjectionMetadata, defaultSource string, defaultFreshness string) projectionEntry {
	return projectionEntry{
		Content:  content,
		Metadata: normalizeProjectionMetadata(meta, defaultSource, defaultFreshness),
	}
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
	spec.ResourcePaths = normalizeWorkflowPaths(spec.ResourcePaths, "/resources")
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
			if strings.HasPrefix(pathValue, "/task_outputs/") {
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

func memoryViewFiles(state sessionState, raw []byte, docs []projectionRecord, resources []projectionRecord, skills []projectionRecord, curated []curatedRecord, events []controlPlaneEvent) map[string]string {
	files := map[string]string{
		"/memory/observations.md": strings.Join(state.Observations, "\n"),
		"/memory/status.json":     string(raw),
		"/memory/summary.md":      renderMemorySummary(state, docs, resources, skills, len(curated), events),
		"/memory/workflows.md":    renderWorkflowSummary(state.Workflows),
		"/memory/projections.md":  renderProjectionSummary(docs, resources, skills),
		"/memory/curated.md":      renderCuratedSummary(curated),
		"/memory/skills_audit.md": renderControlPlaneEventSummary(events, state.ProjectionGeneration),
	}
	workflowRaw, err := json.MarshalIndent(state.Workflows, "", "  ")
	if err == nil {
		files["/memory/workflows.json"] = string(workflowRaw)
	}
	projectionRaw, err := json.MarshalIndent(projectionView{Documents: docs, Resources: resources, Skills: skills}, "", "  ")
	if err == nil {
		files["/memory/projections.json"] = string(projectionRaw)
	}
	curatedRaw, err := json.MarshalIndent(curated, "", "  ")
	if err == nil {
		files["/memory/curated.json"] = string(curatedRaw)
	}
	controlPlaneRaw, err := json.MarshalIndent(controlPlaneEventViews(events, state.ProjectionGeneration), "", "  ")
	if err == nil {
		files["/memory/skills_audit.json"] = string(controlPlaneRaw)
	}
	if len(state.ReadResources) > 0 {
		files["/memory/resources.md"] = strings.Join(state.ReadResources, "\n")
	}
	if len(state.ReadSkills) > 0 {
		files["/memory/skills.md"] = strings.Join(state.ReadSkills, "\n")
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

func renderWorkflowSummary(workflows []workflowView) string {
	lines := []string{"# Managed Workflows", ""}
	if len(workflows) == 0 {
		lines = append(lines, "- no workflows configured")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, workflow := range workflows {
		lines = append(lines, fmt.Sprintf("- [%s] %s (%s)", workflow.Status, workflow.Title, workflow.ID))
		if workflow.StatusSource != "" {
			lines = append(lines, fmt.Sprintf("  source: %s", workflow.StatusSource))
		}
		if workflow.StatusReason != "" {
			lines = append(lines, fmt.Sprintf("  reason: %s", workflow.StatusReason))
		}
		if workflow.Summary != "" {
			lines = append(lines, fmt.Sprintf("  summary: %s", workflow.Summary))
		}
		if len(workflow.ResourcePaths) > 0 {
			lines = append(lines, fmt.Sprintf("  resources: %s", strings.Join(workflow.ResourcePaths, ", ")))
		}
		if len(workflow.ExpectedOutputs) > 0 {
			lines = append(lines, fmt.Sprintf("  outputs: %s", strings.Join(workflow.ExpectedOutputs, ", ")))
		}
		if len(workflow.Evidence) > 0 {
			lines = append(lines, fmt.Sprintf("  evidence: %s", strings.Join(workflow.Evidence, ", ")))
		}
	}
	return strings.Join(lines, "\n") + "\n"
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
			line := fmt.Sprintf("- %s [%s/%s]", record.Path, record.Source, record.Freshness)
			if record.Materialization.State != "" && record.Materialization.State != projectionMaterializationMaterialized {
				line += fmt.Sprintf(" materialization=%s", record.Materialization.State)
				if record.Materialization.Reason != "" {
					line += fmt.Sprintf(" reason=%s", record.Materialization.Reason)
				}
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	if len(resources) > 0 {
		lines = append(lines, "## Resources")
		for _, record := range resources {
			line := fmt.Sprintf("- %s [%s/%s]", record.Path, record.Source, record.Freshness)
			if record.Materialization.State != "" && record.Materialization.State != projectionMaterializationMaterialized {
				line += fmt.Sprintf(" materialization=%s", record.Materialization.State)
				if record.Materialization.Reason != "" {
					line += fmt.Sprintf(" reason=%s", record.Materialization.Reason)
				}
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	if len(skills) > 0 {
		lines = append(lines, "## Skills")
		for _, record := range skills {
			line := fmt.Sprintf("- %s [%s/%s]", record.Path, record.Source, record.Freshness)
			if record.Materialization.State != "" && record.Materialization.State != projectionMaterializationMaterialized {
				line += fmt.Sprintf(" materialization=%s", record.Materialization.State)
				if record.Materialization.Reason != "" {
					line += fmt.Sprintf(" reason=%s", record.Materialization.Reason)
				}
			}
			if record.Eligibility != nil {
				line += fmt.Sprintf(" eligibility=%s", record.Eligibility.State)
			}
			if record.Precedence != nil {
				line += fmt.Sprintf(" precedence=%s:%d", record.Precedence.Tier, record.Precedence.Rank)
			}
			if record.Selection != nil {
				line += fmt.Sprintf(" selection=%s:%s", record.Selection.Mode, record.Selection.State)
				if record.Selection.Scope != "" {
					line += fmt.Sprintf(" scope=%s", record.Selection.Scope)
				}
				if record.Selection.Reason != "" {
					line += fmt.Sprintf(" reason=%s", record.Selection.Reason)
				}
				if record.Selection.WinnerPath != "" {
					line += fmt.Sprintf(" winner=%s", record.Selection.WinnerPath)
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

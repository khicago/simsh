package reference

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/khicago/simsh/pkg/contract"
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
	ProjectionBuildMS    int64           `json:"projection_build_ms,omitempty"`
	ControlPlaneEvents   int             `json:"control_plane_events,omitempty"`
	LastControlPlaneKind string          `json:"last_control_plane_kind,omitempty"`
}

type projectionMetricsView struct {
	ProjectionGeneration     int            `json:"projection_generation"`
	ProjectionBuildMS        int64          `json:"projection_build_ms"`
	ProjectionCounts         map[string]int `json:"projection_counts"`
	FreshnessCounts          map[string]int `json:"freshness_counts,omitempty"`
	MaterializationCounts    map[string]int `json:"materialization_counts,omitempty"`
	ControlPlaneEvents       int            `json:"control_plane_events"`
	UniqueDeniedPaths        int            `json:"unique_denied_paths"`
	CacheHitMetricsAvailable bool           `json:"cache_hit_metrics_available"`
}

type denialView struct {
	ProjectionGeneration int            `json:"projection_generation"`
	UniqueDeniedPaths    int            `json:"unique_denied_paths"`
	ByNamespace          map[string]int `json:"by_namespace"`
	SamplePaths          []string       `json:"sample_paths,omitempty"`
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

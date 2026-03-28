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
	Workflows        []WorkflowSpec
}

type ProjectionMetadata struct {
	Source    string `json:"source,omitempty"`
	Freshness string `json:"freshness,omitempty"`
}

type WorkflowSpec struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary,omitempty"`
	ResourcePaths   []string `json:"resource_paths,omitempty"`
	ExpectedOutputs []string `json:"expected_outputs,omitempty"`
}

type Adapter struct {
	mu              sync.RWMutex
	documents       map[string]projectionEntry
	resources       map[string]projectionEntry
	workflows       []WorkflowSpec
	projectionError error
}

type projectionEntry struct {
	Content  string             `json:"content"`
	Metadata ProjectionMetadata `json:"metadata"`
}

type projectionRecord struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	Freshness string `json:"freshness"`
}

type projectionView struct {
	Documents []projectionRecord `json:"documents,omitempty"`
	Resources []projectionRecord `json:"resources,omitempty"`
}

type sessionState struct {
	Observations   []string       `json:"observations"`
	Freshness      string         `json:"freshness"`
	ReadRefs       []string       `json:"read_refs,omitempty"`
	ReadResources  []string       `json:"read_resources,omitempty"`
	WrittenOutputs []string       `json:"written_outputs,omitempty"`
	DeniedPaths    []string       `json:"denied_paths,omitempty"`
	Workflows      []workflowView `json:"workflows,omitempty"`
}

type workflowView struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary,omitempty"`
	ResourcePaths   []string `json:"resource_paths,omitempty"`
	ExpectedOutputs []string `json:"expected_outputs,omitempty"`
	Status          string   `json:"status"`
	Evidence        []string `json:"evidence,omitempty"`
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
	workflows := make([]WorkflowSpec, 0, len(opts.Workflows))
	for _, workflow := range opts.Workflows {
		if normalized, ok := normalizeWorkflowSpec(workflow); ok {
			workflows = append(workflows, normalized)
		}
	}
	return &Adapter{
		documents: documents,
		resources: resources,
		workflows: workflows,
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
	a.documents[normalizeDocumentName(name)] = projectionEntry{
		Content:  content,
		Metadata: normalizeProjectionMetadata(meta, "control_plane", "updated"),
	}
}

func (a *Adapter) UpdateResource(name string, content string) {
	a.UpsertResource(name, content, ProjectionMetadata{Source: "control_plane", Freshness: "updated"})
}

func (a *Adapter) UpsertResource(name string, content string, meta ProjectionMetadata) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resources[normalizeResourceName(name)] = projectionEntry{
		Content:  content,
		Metadata: normalizeProjectionMetadata(meta, "control_plane", "updated"),
	}
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

func (a *Adapter) SetProjectionError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectionError = err
}

func (a *Adapter) buildProjection(state sessionState) (contract.AdapterProjection, error) {
	if state.Workflows == nil {
		state.Workflows = a.workflowViews(state)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	docs := a.documentRecords()
	resources := a.resourceRecords()
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
	memoryMount, err := mount.NewStaticMount("/memory", "memory", memoryViewFiles(state, raw, docs, resources))
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
		}}
	}
	for name, entry := range a.documents {
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
		files["/resources/"+name] = entry.Content
	}
	if raw, err := json.MarshalIndent(records, "", "  "); err == nil {
		files["/resources/_index.json"] = string(raw)
	}
	return mount.NewStaticMount("/resources", "resource", files)
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

func (a *Adapter) stateFromSession(session contract.Session, freshness string) sessionState {
	state := sessionState{
		Freshness: freshness,
		Workflows: a.workflowViews(sessionState{}),
	}
	if raw := session.State.Opaque[a.AdapterID()]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &state)
		state.Freshness = freshness
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

func normalizeProjectionMetadata(meta ProjectionMetadata, defaultSource string, defaultFreshness string) ProjectionMetadata {
	source := strings.TrimSpace(meta.Source)
	if source == "" {
		source = defaultSource
	}
	freshness := strings.TrimSpace(meta.Freshness)
	if freshness == "" {
		freshness = defaultFreshness
	}
	return ProjectionMetadata{
		Source:    source,
		Freshness: freshness,
	}
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
			Path:      path.Join(root, name),
			Name:      name,
			Namespace: namespace,
			Kind:      kind,
			Source:    entry.Metadata.Source,
			Freshness: entry.Metadata.Freshness,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func memoryViewFiles(state sessionState, raw []byte, docs []projectionRecord, resources []projectionRecord) map[string]string {
	files := map[string]string{
		"/memory/observations.md": strings.Join(state.Observations, "\n"),
		"/memory/status.json":     string(raw),
		"/memory/summary.md":      renderMemorySummary(state, docs, resources),
		"/memory/workflows.md":    renderWorkflowSummary(state.Workflows),
		"/memory/projections.md":  renderProjectionSummary(docs, resources),
	}
	workflowRaw, err := json.MarshalIndent(state.Workflows, "", "  ")
	if err == nil {
		files["/memory/workflows.json"] = string(workflowRaw)
	}
	projectionRaw, err := json.MarshalIndent(projectionView{Documents: docs, Resources: resources}, "", "  ")
	if err == nil {
		files["/memory/projections.json"] = string(projectionRaw)
	}
	if len(state.ReadResources) > 0 {
		files["/memory/resources.md"] = strings.Join(state.ReadResources, "\n")
	}
	return files
}

func renderMemorySummary(state sessionState, docs []projectionRecord, resources []projectionRecord) string {
	lines := []string{
		"# Managed Memory",
		"",
		fmt.Sprintf("- freshness: %s", state.Freshness),
		fmt.Sprintf("- projections.documents: %d", len(docs)),
		fmt.Sprintf("- projections.resources: %d", len(resources)),
		fmt.Sprintf("- observations: %d", len(state.Observations)),
		fmt.Sprintf("- reference_reads: %d", len(state.ReadRefs)),
		fmt.Sprintf("- resource_reads: %d", len(state.ReadResources)),
		fmt.Sprintf("- written_outputs: %d", len(state.WrittenOutputs)),
		fmt.Sprintf("- denied_paths: %d", len(state.DeniedPaths)),
	}
	if len(state.Workflows) > 0 {
		lines = append(lines, "", "## Workflows")
		for _, workflow := range state.Workflows {
			lines = append(lines, fmt.Sprintf("- [%s] %s (%s)", workflow.Status, workflow.Title, workflow.ID))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderWorkflowSummary(workflows []workflowView) string {
	lines := []string{"# Managed Workflows", ""}
	if len(workflows) == 0 {
		lines = append(lines, "- no workflows configured")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, workflow := range workflows {
		lines = append(lines, fmt.Sprintf("- [%s] %s (%s)", workflow.Status, workflow.Title, workflow.ID))
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

func renderProjectionSummary(docs []projectionRecord, resources []projectionRecord) string {
	lines := []string{"# Projection Index", ""}
	if len(docs) == 0 && len(resources) == 0 {
		lines = append(lines, "- no projected items")
		return strings.Join(lines, "\n") + "\n"
	}
	if len(docs) > 0 {
		lines = append(lines, "## Documents")
		for _, record := range docs {
			lines = append(lines, fmt.Sprintf("- %s [%s/%s]", record.Path, record.Source, record.Freshness))
		}
		lines = append(lines, "")
	}
	if len(resources) > 0 {
		lines = append(lines, "## Resources")
		for _, record := range resources {
			lines = append(lines, fmt.Sprintf("- %s [%s/%s]", record.Path, record.Source, record.Freshness))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

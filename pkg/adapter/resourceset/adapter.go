package resourceset

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

const (
	defaultResourceSource    = "resourceset"
	defaultResourceFreshness = "snapshot"
)

type Options struct {
	Resources        map[string]string
	ResourceMetadata map[string]ResourceMetadata
}

type ResourceMetadata struct {
	Source    string `json:"source,omitempty"`
	Freshness string `json:"freshness,omitempty"`
}

type resourceDetail struct {
	Name      string
	Content   string
	Source    string
	Freshness string
}

type resourceRecord struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Source    string `json:"source,omitempty"`
	Freshness string `json:"freshness,omitempty"`
}

type Adapter struct {
	mu            sync.RWMutex
	resources     map[string]resourceDetail
	projectionSeq int
}

type sessionState struct {
	Phase         string   `json:"phase"`
	Generation    int      `json:"generation"`
	ResourceCount int      `json:"resource_count"`
	Observations  []string `json:"observations,omitempty"`
}

func New(opts Options) *Adapter {
	details := map[string]resourceDetail{}
	metadata := normalizeResourceMetadata(opts.ResourceMetadata)
	for name, content := range opts.Resources {
		normalized := normalizeResourceName(name)
		detail := resourceDetail{
			Name:    normalized,
			Content: content,
		}
		if meta, ok := metadata[normalized]; ok {
			detail.Source = meta.Source
			detail.Freshness = meta.Freshness
		}
		if detail.Source == "" {
			detail.Source = defaultResourceSource
		}
		if detail.Freshness == "" {
			detail.Freshness = defaultResourceFreshness
		}
		details[normalized] = detail
	}
	return &Adapter{
		resources: details,
	}
}

func (a *Adapter) AdapterID() string {
	return "resourceset"
}

func (a *Adapter) CreateSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	state := a.stateFromSession(session, "created")
	return a.buildProjection(state)
}

func (a *Adapter) ResumeSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	state := a.stateFromSession(session, "resumed")
	return a.buildProjection(state)
}

func (a *Adapter) ObserveExecution(ctx context.Context, session contract.Session, result contract.ExecutionResult) (contract.AdapterProjection, error) {
	_ = ctx
	state := a.stateFromSession(session, "observed")
	state.Observations = dedupeSorted(append(state.Observations, summarizeTrace(result.Trace)...))
	return a.buildProjection(state)
}

func (a *Adapter) CheckpointSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	state := a.stateFromSession(session, "checkpointed")
	return a.buildProjection(state)
}

func (a *Adapter) CloseSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	state := a.stateFromSession(session, "closed")
	return a.buildProjection(state)
}

func (a *Adapter) buildProjection(state sessionState) (contract.AdapterProjection, error) {
	records, files := a.resourceFilesAndRecords()
	state.Generation = a.nextProjectionGeneration()
	state.ResourceCount = len(records)
	rawState, err := json.Marshal(state)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	resourceMount, err := mount.NewStaticMount("/resources", "resource", files)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	memoryMount, err := mount.NewStaticMount("/memory", "memory", memoryFiles(state, records))
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	return contract.AdapterProjection{
		VirtualMounts: []contract.VirtualMount{resourceMount},
		Memory: contract.MemoryProjection{
			Mount:     memoryMount,
			Freshness: state.Phase,
		},
		OpaqueState: rawState,
	}, nil
}

func (a *Adapter) stateFromSession(session contract.Session, phase string) sessionState {
	state := sessionState{Phase: phase}
	if session.State.Opaque != nil {
		if raw := session.State.Opaque[a.AdapterID()]; len(raw) > 0 {
			var previous sessionState
			if err := json.Unmarshal(raw, &previous); err == nil {
				state = previous
			}
		}
	}
	state.Phase = phase
	return state
}

func (a *Adapter) resourceFilesAndRecords() ([]resourceRecord, map[string]string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	records := make([]resourceRecord, 0, len(a.resources))
	files := map[string]string{}
	for _, detail := range a.resources {
		record := resourceRecord{
			Path:      path.Join("/resources", detail.Name),
			Name:      detail.Name,
			Source:    detail.Source,
			Freshness: detail.Freshness,
		}
		records = append(records, record)
		files[record.Path] = detail.Content
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Path < records[j].Path
	})
	indexRaw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		indexRaw = []byte("[]")
	}
	files["/resources/_index.json"] = string(indexRaw) + "\n"
	return records, files
}

func (a *Adapter) nextProjectionGeneration() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectionSeq++
	return a.projectionSeq
}

func memoryFiles(state sessionState, records []resourceRecord) map[string]string {
	return map[string]string{
		"/memory/summary.md":      renderSummary(state),
		"/memory/observations.md": renderObservations(state),
		"/memory/resources.json":  renderResourceRecords(records),
	}
}

func renderSummary(state sessionState) string {
	lines := []string{
		"# Resource Set Memory",
		"",
		fmt.Sprintf("- phase: %s", state.Phase),
		fmt.Sprintf("- generation: %d", state.Generation),
		fmt.Sprintf("- resource_count: %d", state.ResourceCount),
		fmt.Sprintf("- observations: %d", len(state.Observations)),
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderObservations(state sessionState) string {
	lines := []string{"# Observations", ""}
	if len(state.Observations) == 0 {
		lines = append(lines, "- none recorded yet")
	} else {
		for _, obs := range state.Observations {
			lines = append(lines, fmt.Sprintf("- %s", obs))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderResourceRecords(records []resourceRecord) string {
	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "[]\n"
	}
	return string(raw) + "\n"
}

func summarizeTrace(trace contract.ExecutionTrace) []string {
	lines := make([]string, 0)
	for _, pathValue := range trace.ReadPaths {
		if strings.HasPrefix(pathValue, "/resources/") {
			lines = append(lines, fmt.Sprintf("read:%s", pathValue))
		}
	}
	for _, pathValue := range trace.WrittenPaths {
		if strings.HasPrefix(pathValue, "/resources/") {
			lines = append(lines, fmt.Sprintf("write:%s", pathValue))
		}
	}
	for _, pathValue := range trace.DeniedPaths {
		lines = append(lines, fmt.Sprintf("denied:%s", pathValue))
	}
	return lines
}

func dedupeSorted(lines []string) []string {
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

func normalizeResourceMetadata(metadata map[string]ResourceMetadata) map[string]ResourceMetadata {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]ResourceMetadata, len(metadata))
	for key, value := range metadata {
		name := normalizeResourceName(key)
		if name == "" {
			continue
		}
		out[name] = ResourceMetadata{
			Source:    strings.TrimSpace(value.Source),
			Freshness: strings.TrimSpace(value.Freshness),
		}
	}
	return out
}

func normalizeResourceName(name string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		return "unnamed.txt"
	}
	clean := strings.TrimPrefix(path.Clean("/"+value), "/")
	if clean == "" || clean == "." {
		return "unnamed.txt"
	}
	return clean
}

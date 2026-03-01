package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

type Options struct {
	Documents map[string]string
}

type Adapter struct {
	mu              sync.RWMutex
	documents       map[string]string
	projectionError error
}

type sessionState struct {
	Observations []string `json:"observations"`
	Freshness    string   `json:"freshness"`
}

func New(opts Options) *Adapter {
	documents := map[string]string{}
	for key, value := range opts.Documents {
		documents[normalizeDocumentName(key)] = value
	}
	return &Adapter{documents: documents}
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
	state.Observations = append(state.Observations, summarizeTrace(result.Trace)...)
	state.Observations = dedupeLines(state.Observations)
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
	a.mu.Lock()
	defer a.mu.Unlock()
	a.documents[normalizeDocumentName(name)] = content
}

func (a *Adapter) SetProjectionError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectionError = err
}

func (a *Adapter) buildProjection(state sessionState) (contract.AdapterProjection, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	knowledgeMount, err := a.knowledgeMount()
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	memoryMount, err := mount.NewStaticMount("/memory", "memory", map[string]string{
		"/memory/observations.md": strings.Join(state.Observations, "\n"),
		"/memory/status.json":     string(raw),
	})
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	return contract.AdapterProjection{
		VirtualMounts: []contract.VirtualMount{knowledgeMount},
		Memory: contract.MemoryProjection{
			Mount:     memoryMount,
			Freshness: state.Freshness,
		},
		OpaqueState: raw,
	}, nil
}

func (a *Adapter) knowledgeMount() (contract.VirtualMount, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.projectionError != nil {
		return nil, a.projectionError
	}
	files := make(map[string]string, len(a.documents))
	for name, content := range a.documents {
		files["/knowledge_base/reference/"+name] = content
	}
	if len(files) == 0 {
		files["/knowledge_base/reference/empty.md"] = "# Empty\n"
	}
	return mount.NewStaticMount("/knowledge_base/reference", "reference", files)
}

func (a *Adapter) stateFromSession(session contract.Session, freshness string) sessionState {
	state := sessionState{Freshness: freshness}
	if raw := session.State.Opaque[a.AdapterID()]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &state)
		state.Freshness = freshness
	}
	return state
}

func summarizeTrace(trace contract.ExecutionTrace) []string {
	lines := make([]string, 0)
	for _, pathValue := range trace.ReadPaths {
		if strings.HasPrefix(pathValue, "/knowledge_base/reference/") {
			lines = append(lines, fmt.Sprintf("read-ref:%s", pathValue))
		}
	}
	for _, pathValue := range trace.WrittenPaths {
		lines = append(lines, fmt.Sprintf("wrote:%s", pathValue))
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
	value := strings.TrimSpace(name)
	if value == "" {
		return "empty.md"
	}
	if !strings.HasSuffix(value, ".md") {
		value += ".md"
	}
	return value
}

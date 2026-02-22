package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
)

type Config struct {
	DefaultHostRoot string
	DefaultProfile  string
	DefaultPolicy   string
	DefaultRCFiles  []string
	EnableTestMount bool
}

type executeRequest struct {
	Command     string `json:"command"`
	HostRoot    string `json:"host_root,omitempty"`
	RootDir     string `json:"root_dir,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Policy      string `json:"policy,omitempty"`
	IncludeMeta bool   `json:"include_meta,omitempty"`
}

type pathMeta struct {
	Mode         string   `json:"mode"`
	Access       string   `json:"access"`
	Kind         string   `json:"kind"`
	Lines        int      `json:"lines"`
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
}

type executeMeta struct {
	Paths []pathMeta `json:"paths"`
}

type executeResponse struct {
	Output   string       `json:"output"`
	ExitCode int          `json:"exit_code"`
	Meta     *executeMeta `json:"meta,omitempty"`
}

func NewHandler(cfg Config) http.Handler {
	defaultProfile := strings.TrimSpace(cfg.DefaultProfile)
	if defaultProfile == "" {
		defaultProfile = string(contract.DefaultProfile())
	}
	defaultPolicy := strings.TrimSpace(cfg.DefaultPolicy)
	if defaultPolicy == "" {
		defaultPolicy = string(contract.WriteModeReadOnly)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var req executeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		command := strings.TrimSpace(req.Command)
		if command == "" {
			writeJSON(w, executeResponse{Output: "execute: command is required", ExitCode: contract.ExitCodeUsage})
			return
		}

		hostRoot := strings.TrimSpace(req.HostRoot)
		if hostRoot == "" {
			hostRoot = strings.TrimSpace(req.RootDir)
		}
		if hostRoot == "" {
			hostRoot = cfg.DefaultHostRoot
		}

		profileName := strings.TrimSpace(req.Profile)
		if profileName == "" {
			profileName = defaultProfile
		}
		profile, err := contract.ParseProfile(profileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		policyName := strings.TrimSpace(req.Policy)
		if policyName == "" {
			policyName = defaultPolicy
		}
		policy, err := contract.PolicyPreset(policyName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		env, err := runtimeengine.New(runtimeengine.Options{
			HostRoot:         hostRoot,
			Profile:          profile,
			Policy:           policy,
			RCFiles:          append([]string(nil), cfg.DefaultRCFiles...),
			EnableTestCorpus: cfg.EnableTestMount,
		})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				status = http.StatusRequestTimeout
			}
			http.Error(w, err.Error(), status)
			return
		}

		out, code := env.Execute(r.Context(), command)
		resp := executeResponse{Output: out, ExitCode: code}
		if req.IncludeMeta {
			resp.Meta = buildExecuteMeta(r.Context(), env, command)
		}
		writeJSON(w, resp)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, payload executeResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func buildExecuteMeta(ctx context.Context, env *runtimeengine.Stack, command string) *executeMeta {
	if env == nil {
		return nil
	}
	ops := env.Ops()
	paths := extractAbsPaths(command, ops.RequireAbsolutePath)
	if len(paths) == 0 {
		return nil
	}
	out := make([]pathMeta, 0, len(paths))
	for _, p := range paths {
		row, ok := describePathMeta(ctx, ops, p)
		if !ok {
			row, ok = describePathMetaViaLS(ctx, env, p)
		}
		if ok {
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return &executeMeta{Paths: out}
}

func describePathMeta(ctx context.Context, ops contract.Ops, pathValue string) (pathMeta, bool) {
	if ops.DescribePath == nil {
		return pathMeta{}, false
	}
	described, err := ops.DescribePath(ctx, pathValue)
	if err != nil {
		return pathMeta{}, false
	}

	access := strings.TrimSpace(described.Access)
	if access == "" {
		access = contract.PathAccessReadOnly
		if ops.Policy.AllowWrite() {
			access = contract.PathAccessReadWrite
		}
	}
	access = contract.NormalizePathAccess(access)

	caps := described.Capabilities
	if len(caps) == 0 {
		if described.IsDir {
			caps = []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch}
		} else {
			caps = []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead}
		}
	}
	caps = contract.NormalizePathCapabilities(caps)

	// Runtime policy is an override layer on top of adapter metadata.
	if !ops.Policy.AllowWrite() {
		access = contract.PathAccessReadOnly
	}
	if access == contract.PathAccessReadOnly {
		caps = contract.StripWriteCapabilities(caps)
	}

	kind := strings.TrimSpace(described.Kind)
	if kind == "" {
		if described.IsDir {
			kind = "dir"
		} else {
			kind = "file"
		}
	}

	mode := "-"
	if described.IsDir {
		mode = "d"
	}

	return pathMeta{
		Mode:         mode,
		Access:       access,
		Kind:         kind,
		Lines:        described.LineCount,
		Path:         pathValue,
		Capabilities: caps,
	}, true
}

func describePathMetaViaLS(ctx context.Context, env *runtimeengine.Stack, pathValue string) (pathMeta, bool) {
	raw, code := env.Execute(ctx, "ls -al --fmt json "+pathValue)
	if code != 0 || strings.TrimSpace(raw) == "" {
		return pathMeta{}, false
	}
	var payload struct {
		Columns []string   `json:"columns"`
		Entries []pathMeta `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return pathMeta{}, false
	}
	for _, row := range payload.Entries {
		if strings.TrimSpace(row.Path) == pathValue {
			return row, true
		}
	}
	return pathMeta{}, false
}

func extractAbsPaths(command string, normalize func(string) (string, error)) []string {
	parts := splitShellWords(command)
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, token := range parts {
		token = strings.TrimSpace(token)
		if token == "" || !strings.HasPrefix(token, "/") {
			continue
		}
		if normalize != nil {
			pathValue, err := normalize(token)
			if err == nil {
				token = pathValue
			}
		}
		if strings.TrimSpace(token) == "/" {
			// Skip root unless explicitly passed (common in shells to imply cwd).
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
		if len(out) >= 50 {
			break
		}
	}
	sort.Strings(out)
	return out
}

func splitShellWords(input string) []string {
	tokens := make([]string, 0)
	var buf strings.Builder
	inSingle := false
	inDouble := false
	escape := false

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		tokens = append(tokens, buf.String())
		buf.Reset()
	}

	isSep := func(r rune) bool {
		switch r {
		case '|', '&', ';', '(', ')', '<', '>':
			return true
		default:
			return false
		}
	}

	for _, r := range input {
		if escape {
			buf.WriteRune(r)
			escape = false
			continue
		}
		if inSingle {
			if r == '\'' {
				inSingle = false
				continue
			}
			buf.WriteRune(r)
			continue
		}
		if inDouble {
			switch r {
			case '"':
				inDouble = false
			case '\\':
				escape = true
			default:
				buf.WriteRune(r)
			}
			continue
		}

		switch r {
		case '\\':
			escape = true
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		default:
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
				flush()
				continue
			}
			if isSep(r) {
				flush()
				continue
			}
			buf.WriteRune(r)
		}
	}
	if escape {
		buf.WriteRune('\\')
	}
	flush()
	return tokens
}

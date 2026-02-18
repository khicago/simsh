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
	paths := extractAbsPaths(command)
	if len(paths) == 0 || env == nil {
		return nil
	}
	out := make([]pathMeta, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}

		// Use the runtime itself to derive SSOT access metadata, so mounts/synthetic
		// dirs are handled consistently with the engine's normalized ops.
		raw, code := env.Execute(ctx, "ls -al --fmt json "+p)
		if code != 0 || strings.TrimSpace(raw) == "" {
			continue
		}
		var payload struct {
			Columns []string   `json:"columns"`
			Entries []pathMeta `json:"entries"`
		}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			continue
		}
		for _, row := range payload.Entries {
			if strings.TrimSpace(row.Path) == p {
				out = append(out, row)
				break
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return &executeMeta{Paths: out}
}

func extractAbsPaths(command string) []string {
	parts := strings.Fields(command)
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, raw := range parts {
		token := strings.TrimSpace(raw)
		token = strings.Trim(token, "\"'")
		token = strings.TrimLeft(token, "(")
		token = strings.TrimRight(token, ");,:|&")
		for strings.HasSuffix(token, "&&") || strings.HasSuffix(token, "||") {
			token = strings.TrimRight(token, "&|")
		}
		if !strings.HasPrefix(token, "/") {
			continue
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

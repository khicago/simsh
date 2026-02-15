package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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
	Command  string `json:"command"`
	HostRoot string `json:"host_root,omitempty"`
	RootDir  string `json:"root_dir,omitempty"`
	Profile  string `json:"profile,omitempty"`
	Policy   string `json:"policy,omitempty"`
}

type executeResponse struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
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
		writeJSON(w, executeResponse{Output: out, ExitCode: code})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, payload executeResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestExecuteHandler(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := map[string]any{"command": "env PATH"}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out struct {
		ExecutionID string `json:"execution_id"`
		Output      string `json:"output"`
		Stdout      string `json:"stdout"`
		Stderr      string `json:"stderr"`
		ExitCode    int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", out)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d output=%q", out.ExitCode, out.Output)
	}
	if out.Stdout != out.Output || out.Stderr != "" {
		t.Fatalf("unexpected structured stdout/stderr: %+v", out)
	}
	if !strings.Contains(out.Output, "PATH=/sys/bin:/bin") {
		t.Fatalf("unexpected output %q", out.Output)
	}
}

func TestExecuteHandlerDefaultRCFiles(t *testing.T) {
	tmp := t.TempDir()
	rcPath := filepath.Join(tmp, "task_outputs", "simshrc")
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(rcPath, []byte("export HTTP_BOOT=enabled\n"), 0o644); err != nil {
		t.Fatalf("write rc file failed: %v", err)
	}

	h := NewHandler(Config{
		DefaultHostRoot: tmp,
		DefaultProfile:  "core-strict",
		DefaultPolicy:   "read-only",
		DefaultRCFiles:  []string{"/task_outputs/simshrc"},
	})
	ts := httptest.NewServer(h)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{"command": "env HTTP_BOOT"})
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != 0 || strings.TrimSpace(out.Output) != "HTTP_BOOT=enabled" {
		t.Fatalf("expected rc export in env output: code=%d out=%q", out.ExitCode, out.Output)
	}
}

func TestExecuteHandlerPolicyValidation(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp})
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := map[string]any{"command": "env", "policy": "unknown"}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
}

func TestExecuteHandlerProfileValidation(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp})
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := map[string]any{"command": "env", "profile": "unknown"}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
}

func TestExecuteHandlerWithRootOverride(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: rootA})
	ts := httptest.NewServer(h)
	defer ts.Close()

	virtualPath := "/task_outputs/hello.txt"
	hostFile := filepath.Join(rootB, "task_outputs", "hello.txt")
	if err := os.MkdirAll(filepath.Dir(hostFile), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(hostFile, []byte("line\n"), 0o644); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	payload := map[string]any{"command": "cat -n " + virtualPath, "host_root": rootB}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d output=%q", out.ExitCode, out.Output)
	}
	if !strings.Contains(out.Output, "1:line") {
		t.Fatalf("unexpected output %q", out.Output)
	}
}

func TestExecuteHandlerProfileCapabilities(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	bodyStrict, _ := json.Marshal(map[string]any{"command": "date +%F"})
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(bodyStrict))
	if err != nil {
		t.Fatalf("post strict failed: %v", err)
	}
	defer resp.Body.Close()
	var strictOut struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&strictOut); err != nil {
		t.Fatalf("decode strict failed: %v", err)
	}
	if strictOut.ExitCode == 0 {
		t.Fatalf("expected date blocked in core-strict, got %q", strictOut.Output)
	}

	bodyPlus, _ := json.Marshal(map[string]any{"command": "date +%F", "profile": "bash-plus"})
	resp2, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(bodyPlus))
	if err != nil {
		t.Fatalf("post bash-plus failed: %v", err)
	}
	defer resp2.Body.Close()
	var plusOut struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&plusOut); err != nil {
		t.Fatalf("decode plus failed: %v", err)
	}
	if plusOut.ExitCode != 0 {
		t.Fatalf("expected date enabled in bash-plus, code=%d out=%q", plusOut.ExitCode, plusOut.Output)
	}
}

func TestExecuteHandlerIncludeMeta(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := map[string]any{"command": "ls -l /sys", "include_meta": true}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
		Meta     *struct {
			Paths []struct {
				Path         string   `json:"path"`
				Access       string   `json:"access"`
				Kind         string   `json:"kind"`
				Mode         string   `json:"mode"`
				Capabilities []string `json:"capabilities"`
			} `json:"paths"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d output=%q", out.ExitCode, out.Output)
	}
	if !strings.Contains(out.Output, "/sys/bin") {
		t.Fatalf("expected /sys/bin in output: %q", out.Output)
	}
	if out.Meta == nil || len(out.Meta.Paths) == 0 {
		t.Fatalf("expected meta.paths, got %+v", out.Meta)
	}
	found := false
	for _, p := range out.Meta.Paths {
		if p.Path == "/sys" {
			found = true
			if p.Access != "ro" || p.Mode != "d" || p.Kind == "" {
				t.Fatalf("unexpected /sys meta row: %+v", p)
			}
			if len(p.Capabilities) == 0 {
				t.Fatalf("expected capabilities for /sys: %+v", p)
			}
		}
	}
	if !found {
		t.Fatalf("expected /sys in meta.paths, got %+v", out.Meta.Paths)
	}
}

func TestExecuteHandlerIncludeMetaQuotedPath(t *testing.T) {
	tmp := t.TempDir()
	hostFile := filepath.Join(tmp, "task_outputs", "a b.txt")
	if err := os.MkdirAll(filepath.Dir(hostFile), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(hostFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	payload := map[string]any{"command": `cat "/task_outputs/a b.txt"`, "include_meta": true}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/v1/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
		Meta     *struct {
			Paths []struct {
				Path   string `json:"path"`
				Access string `json:"access"`
			} `json:"paths"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d output=%q", out.ExitCode, out.Output)
	}
	if !strings.Contains(out.Output, "hello") {
		t.Fatalf("unexpected output %q", out.Output)
	}
	if out.Meta == nil || len(out.Meta.Paths) == 0 {
		t.Fatalf("expected meta.paths, got %+v", out.Meta)
	}
	found := false
	for _, p := range out.Meta.Paths {
		if p.Path == "/task_outputs/a b.txt" {
			found = true
			if p.Access != "ro" {
				t.Fatalf("expected read-only access under read-only policy, got %+v", p)
			}
		}
	}
	if !found {
		t.Fatalf("expected quoted path in meta.paths, got %+v", out.Meta.Paths)
	}
}

func TestExtractAbsPathsParsesShellStyleTokens(t *testing.T) {
	paths := extractAbsPaths(
		`cat "/task_outputs/a b.txt" && ls -l /sys/bin;/bin/date >/task_outputs/out.txt /`,
		func(p string) (string, error) { return p, nil },
	)
	sort.Strings(paths)
	expected := []string{"/bin/date", "/sys/bin", "/task_outputs/a b.txt", "/task_outputs/out.txt"}
	if len(paths) != len(expected) {
		t.Fatalf("unexpected paths len=%d paths=%v", len(paths), paths)
	}
	for i := range expected {
		if paths[i] != expected[i] {
			t.Fatalf("unexpected paths[%d]=%q expected=%q all=%v", i, paths[i], expected[i], paths)
		}
	}
}

func TestSessionLifecycleHandler(t *testing.T) {
	tmp := t.TempDir()
	rcPath := filepath.Join(tmp, "task_outputs", "simshrc")
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(rcPath, []byte("export HTTP_SESSION=enabled\n"), 0o644); err != nil {
		t.Fatalf("write rc file failed: %v", err)
	}

	h := NewHandler(Config{
		DefaultHostRoot: tmp,
		DefaultProfile:  "core-strict",
		DefaultPolicy:   "read-only",
		DefaultRCFiles:  []string{"/task_outputs/simshrc"},
	})
	ts := httptest.NewServer(h)
	defer ts.Close()

	createResp := postJSON(t, ts.URL+"/v1/sessions", map[string]any{})
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(createResp.Body)
		t.Fatalf("unexpected create status=%d body=%s", createResp.StatusCode, string(raw))
	}
	var created struct {
		Session struct {
			SessionID string `json:"session_id"`
			State     struct {
				RCFiles []string `json:"rc_files"`
			} `json:"state"`
		} `json:"session"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create failed: %v", err)
	}
	if created.Session.SessionID == "" {
		t.Fatalf("expected session_id in create response")
	}
	if len(created.Session.State.RCFiles) != 1 || created.Session.State.RCFiles[0] != "/task_outputs/simshrc" {
		t.Fatalf("unexpected create rc files: %v", created.Session.State.RCFiles)
	}

	executeResp := postJSON(t, ts.URL+"/v1/execute", map[string]any{
		"session_id": created.Session.SessionID,
		"command":    "env HTTP_SESSION",
	})
	defer executeResp.Body.Close()
	if executeResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(executeResp.Body)
		t.Fatalf("unexpected execute status=%d body=%s", executeResp.StatusCode, string(raw))
	}
	var executed struct {
		ExecutionID string `json:"execution_id"`
		Output      string `json:"output"`
		Stdout      string `json:"stdout"`
		ExitCode    int    `json:"exit_code"`
		SessionID   string `json:"session_id"`
	}
	if err := json.NewDecoder(executeResp.Body).Decode(&executed); err != nil {
		t.Fatalf("decode execute failed: %v", err)
	}
	if executed.SessionID != created.Session.SessionID {
		t.Fatalf("unexpected execute session_id=%q want=%q", executed.SessionID, created.Session.SessionID)
	}
	if executed.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", executed)
	}
	if executed.ExitCode != 0 || strings.TrimSpace(executed.Stdout) != "HTTP_SESSION=enabled" {
		t.Fatalf("unexpected execute output: %+v", executed)
	}

	checkpointResp := postJSON(t, ts.URL+"/v1/sessions/"+created.Session.SessionID+"/checkpoint", map[string]any{})
	defer checkpointResp.Body.Close()
	if checkpointResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(checkpointResp.Body)
		t.Fatalf("unexpected checkpoint status=%d body=%s", checkpointResp.StatusCode, string(raw))
	}

	closeResp := postJSON(t, ts.URL+"/v1/sessions/"+created.Session.SessionID+"/close", map[string]any{})
	defer closeResp.Body.Close()
	if closeResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(closeResp.Body)
		t.Fatalf("unexpected close status=%d body=%s", closeResp.StatusCode, string(raw))
	}

	closedExecuteResp := postJSON(t, ts.URL+"/v1/execute", map[string]any{
		"session_id": created.Session.SessionID,
		"command":    "env HTTP_SESSION",
	})
	defer closedExecuteResp.Body.Close()
	if closedExecuteResp.StatusCode != http.StatusConflict {
		raw, _ := io.ReadAll(closedExecuteResp.Body)
		t.Fatalf("unexpected closed execute status=%d body=%s", closedExecuteResp.StatusCode, string(raw))
	}

	if err := os.Remove(rcPath); err != nil {
		t.Fatalf("remove rc failed: %v", err)
	}
	resumeResp := postJSON(t, ts.URL+"/v1/sessions/"+created.Session.SessionID+"/resume", map[string]any{})
	defer resumeResp.Body.Close()
	if resumeResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resumeResp.Body)
		t.Fatalf("unexpected resume status=%d body=%s", resumeResp.StatusCode, string(raw))
	}

	resumeExecuteResp := postJSON(t, ts.URL+"/v1/execute", map[string]any{
		"session_id": created.Session.SessionID,
		"command":    "env HTTP_SESSION",
	})
	defer resumeExecuteResp.Body.Close()
	if resumeExecuteResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resumeExecuteResp.Body)
		t.Fatalf("unexpected resumed execute status=%d body=%s", resumeExecuteResp.StatusCode, string(raw))
	}
	var resumed struct {
		ExecutionID string `json:"execution_id"`
		Output      string `json:"output"`
		Stdout      string `json:"stdout"`
		ExitCode    int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resumeExecuteResp.Body).Decode(&resumed); err != nil {
		t.Fatalf("decode resumed execute failed: %v", err)
	}
	if resumed.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", resumed)
	}
	if resumed.ExitCode != 0 || strings.TrimSpace(resumed.Stdout) != "HTTP_SESSION=enabled" {
		t.Fatalf("unexpected resumed output: %+v", resumed)
	}
}

func TestExecuteHandlerRejectsSessionPolicyEscalation(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	createResp := postJSON(t, ts.URL+"/v1/sessions", map[string]any{})
	defer createResp.Body.Close()
	var created struct {
		Session struct {
			SessionID string `json:"session_id"`
		} `json:"session"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create failed: %v", err)
	}

	resp := postJSON(t, ts.URL+"/v1/execute", map[string]any{
		"session_id": created.Session.SessionID,
		"command":    "echo hi",
		"policy":     "full",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
}

func postJSON(t *testing.T, url string, payload map[string]any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	return resp
}

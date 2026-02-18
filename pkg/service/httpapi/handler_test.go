package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d output=%q", out.ExitCode, out.Output)
	}
	if !strings.Contains(out.Output, "PATH=/sys/bin:/bin") {
		t.Fatalf("unexpected output %q", out.Output)
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

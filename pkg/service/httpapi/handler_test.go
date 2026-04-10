package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
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

func TestSessionHandlerOptionalJSONBody(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	cases := []struct {
		name           string
		body           string
		hasBody        bool
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "empty body uses defaults",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "literal null uses defaults",
			body:           "null",
			hasBody:        true,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid json rejected",
			body:           "{",
			hasBody:        true,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "invalid json body",
		},
		{
			name:           "trailing json rejected",
			body:           "{}{}",
			hasBody:        true,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "invalid json body",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.hasBody {
				body = strings.NewReader(tc.body)
			}
			resp := postBody(t, ts.URL+"/v1/sessions", body)
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				raw, _ := io.ReadAll(resp.Body)
				t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
			}
			if tc.wantStatusCode != http.StatusOK {
				raw, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(raw), tc.wantBody) {
					t.Fatalf("unexpected body %q, want substring %q", string(raw), tc.wantBody)
				}
				return
			}

			var out sessionResponse
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if out.Session.SessionID == "" {
				t.Fatalf("expected session_id in response: %+v", out)
			}
		})
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

func TestExecuteHandlerInvalidJSONBody(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp})
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp := postBody(t, ts.URL+"/v1/execute", strings.NewReader("{"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "invalid json body") {
		t.Fatalf("unexpected body %q", string(raw))
	}
}

func TestExecuteHandlerRejectsTrailingJSONBody(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp})
	ts := httptest.NewServer(h)
	defer ts.Close()

	executePath := "/" + strings.Join([]string{"v1", "execute"}, "/")
	resp := postBody(t, ts.URL+executePath, strings.NewReader(`{"command":"echo hi"}{"command":"echo again"}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "invalid json body") {
		t.Fatalf("unexpected body %q", string(raw))
	}
}

func TestExecuteHandlerRejectsUnknownFields(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp})
	ts := httptest.NewServer(h)
	defer ts.Close()

	executePath := "/" + strings.Join([]string{"v1", "execute"}, "/")
	resp := postBody(t, ts.URL+executePath, strings.NewReader(`{"command":"echo hi","unexpected":true}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "invalid json body") {
		t.Fatalf("unexpected body %q", string(raw))
	}
}

func TestExecuteHandlerCommandRequired(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp})
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/v1/execute", map[string]any{"command": " \n\t "})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		Output   string `json:"output"`
		Stdout   string `json:"stdout"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != contract.ExitCodeUsage {
		t.Fatalf("unexpected exit code %d, want %d", out.ExitCode, contract.ExitCodeUsage)
	}
	if out.Output != "execute: command is required" || out.Stdout != out.Output {
		t.Fatalf("unexpected usage payload: %+v", out)
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

func TestExecuteHandlerIncludeMetaExplicitRootPath(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/v1/execute", map[string]any{
		"command":      "ls -l /",
		"include_meta": true,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		ExitCode int `json:"exit_code"`
		Meta     *struct {
			Paths []pathMeta `json:"paths"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d", out.ExitCode)
	}
	if out.Meta == nil || len(out.Meta.Paths) == 0 {
		t.Fatalf("expected meta.paths, got %+v", out.Meta)
	}

	for _, row := range out.Meta.Paths {
		if row.Path == "/" {
			if row.Mode != "d" || row.Access != contract.PathAccessReadOnly || row.Kind == "" {
				t.Fatalf("unexpected root metadata: %+v", row)
			}
			if len(row.Capabilities) == 0 {
				t.Fatalf("expected capabilities for root metadata: %+v", row)
			}
			return
		}
	}
	t.Fatalf("expected explicit root path in meta.paths, got %+v", out.Meta.Paths)
}

func TestStatusForSessionError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "session not found",
			err:  fmt.Errorf("wrapped: %w", runtimeengine.ErrSessionNotFound),
			want: http.StatusNotFound,
		},
		{
			name: "session closed",
			err:  fmt.Errorf("wrapped: %w", runtimeengine.ErrSessionClosed),
			want: http.StatusConflict,
		},
		{
			name: "policy ceiling",
			err:  fmt.Errorf("wrapped: %w", contract.ErrPolicyCeilingExceeded),
			want: http.StatusBadRequest,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			want: http.StatusRequestTimeout,
		},
		{
			name: "canceled",
			err:  context.Canceled,
			want: http.StatusRequestTimeout,
		},
		{
			name: "default",
			err:  fmt.Errorf("boom"),
			want: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusForSessionError(tc.err); got != tc.want {
				t.Fatalf("statusForSessionError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestDescribePathMetaViaLSMountedPath(t *testing.T) {
	env, err := runtimeengine.New(runtimeengine.Options{
		HostRoot:         t.TempDir(),
		Profile:          contract.ProfileCoreStrict,
		Policy:           contract.DefaultPolicy(),
		EnableTestCorpus: true,
	})
	if err != nil {
		t.Fatalf("runtimeengine.New(...) error = %v", err)
	}

	row, ok := describePathMetaViaLS(context.Background(), env, "/test/core-strict/cases/echo-basic.sh")
	if !ok {
		t.Fatal("describePathMetaViaLS(...) = !ok, want mounted-path metadata")
	}
	if row.Path != "/test/core-strict/cases/echo-basic.sh" {
		t.Fatalf("describePathMetaViaLS(...).Path = %q, want %q", row.Path, "/test/core-strict/cases/echo-basic.sh")
	}
	if row.Access == "" {
		t.Fatalf("describePathMetaViaLS(...).Access empty: %+v", row)
	}
	if row.Kind == "" {
		t.Fatalf("describePathMetaViaLS(...).Kind empty: %+v", row)
	}
	if row.Mode == "" {
		t.Fatalf("describePathMetaViaLS(...).Mode empty: %+v", row)
	}
}

func TestDescribePathMetaDefaults(t *testing.T) {
	cases := []struct {
		name       string
		policy     contract.ExecutionPolicy
		described  contract.PathMeta
		wantAccess string
		wantKind   string
		wantMode   string
		wantCaps   []string
	}{
		{
			name:       "read only dir defaults from policy and shape",
			policy:     contract.DefaultPolicy(),
			described:  contract.PathMeta{Exists: true, IsDir: true},
			wantAccess: contract.PathAccessReadOnly,
			wantKind:   "dir",
			wantMode:   "d",
			wantCaps:   []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
		},
		{
			name:       "write policy defaults file access to rw",
			policy:     contract.ExecutionPolicy{WriteMode: contract.WriteModeWriteLimited},
			described:  contract.PathMeta{Exists: true},
			wantAccess: contract.PathAccessReadWrite,
			wantKind:   "file",
			wantMode:   "-",
			wantCaps:   []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
		},
		{
			name:   "read only strips write capabilities",
			policy: contract.DefaultPolicy(),
			described: contract.PathMeta{
				Exists: true,
				Kind:   "custom",
				Access: contract.PathAccessReadWrite,
				Capabilities: []string{
					contract.PathCapabilityDescribe,
					contract.PathCapabilityRead,
					contract.PathCapabilityWrite,
					contract.PathCapabilityAppend,
					contract.PathCapabilityEdit,
					contract.PathCapabilityMkdir,
					contract.PathCapabilityRemove,
				},
			},
			wantAccess: contract.PathAccessReadOnly,
			wantKind:   "custom",
			wantMode:   "-",
			wantCaps:   []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ops := contract.Ops{
				Policy: tc.policy,
				DescribePath: func(context.Context, string) (contract.PathMeta, error) {
					return tc.described, nil
				},
			}

			got, ok := describePathMeta(context.Background(), ops, "/task_outputs/example")
			if !ok {
				t.Fatal("describePathMeta(...) = !ok, want metadata")
			}
			if got.Access != tc.wantAccess || got.Kind != tc.wantKind || got.Mode != tc.wantMode {
				t.Fatalf("unexpected metadata row: %+v", got)
			}
			if !slices.Equal(got.Capabilities, tc.wantCaps) {
				t.Fatalf("unexpected capabilities %v, want %v", got.Capabilities, tc.wantCaps)
			}
		})
	}
}

func TestExtractAbsPathsParsesShellStyleTokens(t *testing.T) {
	paths := extractAbsPaths(
		`cat "/task_outputs/a b.txt" && ls -l /sys/bin;/bin/date >/task_outputs/out.txt /`,
		func(p string) (string, error) { return p, nil },
	)
	sort.Strings(paths)
	expected := []string{"/", "/bin/date", "/sys/bin", "/task_outputs/a b.txt", "/task_outputs/out.txt"}
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

func TestExecuteHandlerRejectsSessionOverrides(t *testing.T) {
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

	cases := []struct {
		name    string
		payload map[string]any
	}{
		{
			name: "host root override",
			payload: map[string]any{
				"session_id": created.Session.SessionID,
				"command":    "env PATH",
				"host_root":  t.TempDir(),
			},
		},
		{
			name: "root dir override",
			payload: map[string]any{
				"session_id": created.Session.SessionID,
				"command":    "env PATH",
				"root_dir":   t.TempDir(),
			},
		},
		{
			name: "profile override",
			payload: map[string]any{
				"session_id": created.Session.SessionID,
				"command":    "env PATH",
				"profile":    "bash-plus",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := postJSON(t, ts.URL+"/v1/execute", tc.payload)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				raw, _ := io.ReadAll(resp.Body)
				t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
			}
			raw, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(raw), "session-bound execute does not accept host_root/root_dir/profile overrides") {
				t.Fatalf("unexpected body %q", string(raw))
			}
		})
	}
}

func TestSessionHandlerRejectsUnknownFields(t *testing.T) {
	tmp := t.TempDir()
	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	sessionsPath := "/" + strings.Join([]string{"v1", "sessions"}, "/")
	resp := postBody(t, ts.URL+sessionsPath, strings.NewReader(`{"host_root":".","unexpected":true}`))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "invalid json body") {
		t.Fatalf("unexpected body %q", string(raw))
	}
}

func TestExecuteHandlerReturnsTracePaths(t *testing.T) {
	tmp := t.TempDir()
	hostFile := filepath.Join(tmp, "task_outputs", "trace.txt")
	if err := os.MkdirAll(filepath.Dir(hostFile), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(hostFile, []byte("trace\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	h := NewHandler(Config{DefaultHostRoot: tmp, DefaultProfile: "core-strict", DefaultPolicy: "read-only"})
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/v1/execute", map[string]any{"command": "cat /task_outputs/trace.txt"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status=%d body=%s", resp.StatusCode, string(raw))
	}

	var out struct {
		Trace struct {
			Command        string   `json:"command"`
			RequestedPaths []string `json:"requested_paths"`
			ReadPaths      []string `json:"read_paths"`
		} `json:"trace"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.Trace.Command != "cat" {
		t.Fatalf("unexpected command trace: %+v", out.Trace)
	}
	if !containsString(out.Trace.RequestedPaths, "/task_outputs/trace.txt") {
		t.Fatalf("expected requested path in trace: %+v", out.Trace)
	}
	if !containsString(out.Trace.ReadPaths, "/task_outputs/trace.txt") {
		t.Fatalf("expected read path in trace: %+v", out.Trace)
	}
}

func postJSON(t *testing.T, url string, payload map[string]any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	resp, err := postRequest(url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	return resp
}

func postBody(t *testing.T, url string, body io.Reader) *http.Response {
	t.Helper()
	resp, err := postRequest(url, body)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	return resp
}

func postRequest(url string, body io.Reader) (*http.Response, error) {
	return http.Post(url, "application/json", body)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

package fs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/khicago/simsh/pkg/adapter/agentfs"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

type ExternalCallbacks struct {
	ListExternalCommands func(ctx context.Context) ([]contract.ExternalCommand, error)
	RunExternalCommand   func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error)
	ReadExternalManual   func(ctx context.Context, command string) (string, error)
}

type EnvironmentOptions struct {
	HostRoot          string
	Profile           contract.CompatibilityProfile
	Policy            contract.ExecutionPolicy
	PathEnv           []string
	EnableTestCorpus  bool
	ExternalCallbacks ExternalCallbacks
	FormatLSLongRow   contract.LSLongRowFormatter
	AuditSink         contract.AuditSink
	RelevanceEval     agentfs.RelevanceEvaluator
}

func NewAIFilesystem(hostRoot string, relevanceEval agentfs.RelevanceEvaluator) (contract.Filesystem, error) {
	base := strings.TrimSpace(hostRoot)
	if base == "" {
		return agentfs.NewDefaultFilesystem("")
	}
	absRoot, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	return agentfs.NewFilesystem(agentfs.Options{
		Zones: []agentfs.ZoneConfig{
			{
				VirtualRoot: agentfs.VirtualTaskOutputRoot,
				HostRoot:    filepath.Join(absRoot, "task_outputs"),
				Writable:    true,
				Kind:        "task_output_dir",
			},
			{
				VirtualRoot: agentfs.VirtualTempWorkRoot,
				HostRoot:    filepath.Join(absRoot, "temp_work"),
				Writable:    true,
				Kind:        "temp_work_dir",
			},
			{
				VirtualRoot: agentfs.VirtualKnowledgeRoot,
				HostRoot:    filepath.Join(absRoot, "knowledge_base"),
				Writable:    false,
				Kind:        "knowledge_dir",
			},
		},
		EnsureHostRoots: true,
		PathEnv:         []string{agentfs.VirtualTaskOutputRoot, agentfs.VirtualTempWorkRoot, agentfs.VirtualKnowledgeRoot},
		RelevanceEval:   relevanceEval,
	})
}

func NewRuntimeOps(opts EnvironmentOptions) (contract.Ops, error) {
	fsys, err := NewAIFilesystem(opts.HostRoot, opts.RelevanceEval)
	if err != nil {
		return contract.Ops{}, err
	}

	ops := contract.OpsFromFilesystem(fsys)
	if opts.Profile != "" {
		ops.Profile = opts.Profile
	}
	if opts.Policy.WriteMode != "" {
		ops.Policy = opts.Policy
	}
	if len(opts.PathEnv) > 0 {
		ops.PathEnv = append(ops.PathEnv, opts.PathEnv...)
	}
	ops.ListExternalCommands = opts.ExternalCallbacks.ListExternalCommands
	ops.RunExternalCommand = opts.ExternalCallbacks.RunExternalCommand
	ops.ReadExternalManual = opts.ExternalCallbacks.ReadExternalManual
	ops.FormatLSLongRow = opts.FormatLSLongRow
	ops.AuditSink = opts.AuditSink

	if opts.EnableTestCorpus {
		m, err := mount.NewBaselineCorpusMount()
		if err != nil {
			return contract.Ops{}, fmt.Errorf("enable test corpus mount failed: %w", err)
		}
		ops.VirtualMounts = append(ops.VirtualMounts, m)
	}
	return ops, nil
}

func DescribeMarkdown() string {
	lines := []string{
		"## Filesystem Model",
		"- virtual root `/` exposes purpose-oriented directories only:",
		"- `/task_outputs`: durable deliverables and final outputs (writable)",
		"- `/temp_work`: temporary intermediates and disposable artifacts (writable)",
		"- `/knowledge_base`: read-only references and knowledge files",
		"- path policy is zone-scoped and blocks path escape by host-root relative checks",
		"- metadata is AI-oriented: kind, line count, frontmatter rows, speaker-like rows, relevance",
	}
	return strings.Join(lines, "\n")
}

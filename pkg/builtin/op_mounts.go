package builtin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type mountsFormat string

const (
	mountsFormatText mountsFormat = "text"
	mountsFormatJSON mountsFormat = "json"
)

type mountStatusRow struct {
	MountPoint          string                            `json:"mount_point"`
	TruthModel          contract.MountTruthModel          `json:"truth_model,omitempty"`
	MaterializationMode contract.MountMaterializationMode `json:"materialization_mode,omitempty"`
	WriteSemantics      contract.MountWriteSemantics      `json:"write_semantics,omitempty"`
	LatencyClass        contract.MountLatencyClass        `json:"latency_class,omitempty"`
	SupportedCLIClasses []contract.MountCLIClass         `json:"supported_cli_classes,omitempty"`
	Consistency         contract.MountConsistency         `json:"consistency"`
	SLO                 contract.MountSLO                 `json:"slo"`
	HasRefresher        bool                              `json:"has_refresher"`
	HasStatus           bool                              `json:"has_status"`
	HasStats            bool                              `json:"has_stats"`
	RuntimeStatus       contract.MountRuntimeStatus       `json:"runtime_status,omitempty"`
}

func specMounts() engine.CommandSpec {
	return engine.CommandSpec{
		Name:    CommandMounts,
		Summary: "show active mount points, profiles, and optional runtime status evidence",
		Manual:  "mounts [--fmt text|json]",
		Tips: []string{
			"mounts shows the active virtual mount contract surface visible to the runtime.",
			"runtime_status is optional evidence from mounts that expose a read-only status capability.",
			"Use --fmt json for machine-readable mount point/profile/status data.",
		},
		DefaultOutput:    "mount contract summary rows",
		StructuredOutput: "mount contract records",
		StructuredFlags:  []string{"--fmt json"},
		StdinMode:        contract.BuiltinStdinNone,
		Examples:         []string{"mounts", "mounts --fmt json"},
		DetailedManual:   LoadEmbeddedManual("mounts"),
		Run:              runMounts,
	}
}

func runMounts(runtime engine.CommandRuntime, args []string) (string, int) {
	format := mountsFormatText
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return "mounts: --fmt requires one value: text|json", contract.ExitCodeUsage
			}
			idx++
			parsed, ok := parseMountsFormat(args[idx])
			if !ok {
				return fmt.Sprintf("mounts: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage
			}
			format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseMountsFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return fmt.Sprintf("mounts: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage
			}
			format = parsed
		default:
			return fmt.Sprintf("mounts: unsupported flag %s", arg), contract.ExitCodeUsage
		}
	}

	rows := collectMountStatusRows(runtime)
	if format == mountsFormatJSON {
		raw, err := json.MarshalIndent(struct {
			Mounts []mountStatusRow `json:"mounts"`
		}{Mounts: rows}, "", "  ")
		if err != nil {
			return fmt.Sprintf("mounts: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}
	if len(rows) == 0 {
		return "", 0
	}
	lines := make([]string, 0, len(rows)+1)
	for _, row := range rows {
		line := fmt.Sprintf("%s %s %s %s refresh=%t status=%t stats=%t", row.MountPoint, row.TruthModel, row.MaterializationMode, row.LatencyClass, row.HasRefresher, row.HasStatus, row.HasStats)
		if row.RuntimeStatus.Freshness != "" {
			line += " freshness=" + row.RuntimeStatus.Freshness
		}
		if row.RuntimeStatus.Materialization != "" {
			line += " materialization=" + row.RuntimeStatus.Materialization
		}
		if row.RuntimeStatus.StatusError != "" {
			line += " status_error=" + row.RuntimeStatus.StatusError
		}
		lines = append(lines, line)
	}
	lines = append(lines, "# columns: mount_point truth materialization latency refresh status stats [freshness materialization status_error]")
	return strings.Join(lines, "\n"), 0
}

func parseMountsFormat(raw string) (mountsFormat, bool) {
	switch strings.TrimSpace(raw) {
	case "text", "":
		return mountsFormatText, true
	case "json":
		return mountsFormatJSON, true
	default:
		return mountsFormatText, false
	}
}

func collectMountStatusRows(runtime engine.CommandRuntime) []mountStatusRow {
	rows := make([]mountStatusRow, 0, len(runtime.Ops.VirtualMounts))
	for _, mount := range runtime.Ops.VirtualMounts {
		if mount == nil {
			continue
		}
		profile := contract.NormalizeMountProfile(mount.Profile())
		row := mountStatusRow{
			MountPoint:          mount.MountPoint(),
			TruthModel:          profile.TruthModel,
			MaterializationMode: profile.MaterializationMode,
			WriteSemantics:      profile.WriteSemantics,
			LatencyClass:        profile.LatencyClass,
			SupportedCLIClasses: append([]contract.MountCLIClass(nil), profile.SupportedCLIClasses...),
			Consistency:         profile.Consistency,
			SLO:                 profile.SLO,
			HasRefresher:        hasRefresherCapability(mount),
			HasStatus:           hasMountStatusCapability(mount),
			HasStats:            hasStatsCapability(mount),
			RuntimeStatus:       readMountRuntimeStatus(runtime, mount),
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].MountPoint < rows[j].MountPoint })
	return rows
}

func readMountRuntimeStatus(runtime engine.CommandRuntime, mount contract.VirtualMount) contract.MountRuntimeStatus {
	provider, ok := mount.(contract.MountStatusProvider)
	if !ok {
		return contract.MountRuntimeStatus{}
	}
	status, err := provider.MountStatus(runtime.Ctx)
	if err != nil {
		return contract.MountRuntimeStatus{StatusError: strings.TrimSpace(err.Error())}
	}
	return contract.NormalizeMountRuntimeStatus(status)
}

func hasRefresherCapability(mount contract.VirtualMount) bool {
	_, ok := mount.(contract.Refresher)
	return ok
}

func hasMountStatusCapability(mount contract.VirtualMount) bool {
	_, ok := mount.(contract.MountStatusProvider)
	return ok
}

func hasStatsCapability(mount contract.VirtualMount) bool {
	_, ok := mount.(contract.StatsProvider)
	return ok
}

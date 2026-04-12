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
	MountPoint          string                  `json:"mount_point"`
	TruthModel          contract.MountTruthModel `json:"truth_model,omitempty"`
	MaterializationMode contract.MountMaterializationMode `json:"materialization_mode,omitempty"`
	WriteSemantics      contract.MountWriteSemantics `json:"write_semantics,omitempty"`
	LatencyClass        contract.MountLatencyClass `json:"latency_class,omitempty"`
	SupportedCLIClasses []contract.MountCLIClass `json:"supported_cli_classes,omitempty"`
	Consistency         contract.MountConsistency `json:"consistency"`
	SLO                 contract.MountSLO `json:"slo"`
	HasRefresher        bool `json:"has_refresher"`
	HasStats            bool `json:"has_stats"`
}

func specMounts() engine.CommandSpec {
	return engine.CommandSpec{
		Name:    CommandMounts,
		Summary: "show active mount points, profiles, and declared contract capabilities",
		Manual:  "mounts [--fmt text|json]",
		Tips: []string{
			"mounts shows the active virtual mount contract surface visible to the runtime.",
			"Use --fmt json for machine-readable mount point/profile/SLO data.",
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

	rows := collectMountStatusRows(runtime.Ops.VirtualMounts)
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
		lines = append(lines, fmt.Sprintf("%s %s %s %s refresh=%t stats=%t", row.MountPoint, row.TruthModel, row.MaterializationMode, row.LatencyClass, row.HasRefresher, row.HasStats))
	}
	lines = append(lines, "# columns: mount_point truth materialization latency refresh stats")
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

func collectMountStatusRows(mounts []contract.VirtualMount) []mountStatusRow {
	rows := make([]mountStatusRow, 0, len(mounts))
	for _, mount := range mounts {
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
			HasStats:            hasStatsCapability(mount),
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].MountPoint < rows[j].MountPoint })
	return rows
}

func hasRefresherCapability(mount contract.VirtualMount) bool {
	_, ok := mount.(contract.Refresher)
	return ok
}

func hasStatsCapability(mount contract.VirtualMount) bool {
	_, ok := mount.(contract.StatsProvider)
	return ok
}

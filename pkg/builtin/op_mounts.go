package builtin

import (
	"encoding/json"
	"errors"
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

type mountsOptions struct {
	action        string
	format        mountsFormat
	targets       []string
	requireNarrow bool
}

type mountStatusRow struct {
	MountPoint          string                            `json:"mount_point"`
	TruthModel          contract.MountTruthModel          `json:"truth_model,omitempty"`
	MaterializationMode contract.MountMaterializationMode `json:"materialization_mode,omitempty"`
	WriteSemantics      contract.MountWriteSemantics      `json:"write_semantics,omitempty"`
	LatencyClass        contract.MountLatencyClass        `json:"latency_class,omitempty"`
	SupportedCLIClasses []contract.MountCLIClass          `json:"supported_cli_classes,omitempty"`
	Consistency         contract.MountConsistency         `json:"consistency"`
	SLO                 contract.MountSLO                 `json:"slo"`
	HasRefresher        bool                              `json:"has_refresher"`
	HasStatus           bool                              `json:"has_status"`
	HasStats            bool                              `json:"has_stats"`
	RuntimeStatus       contract.MountRuntimeStatus       `json:"runtime_status,omitempty"`
}

type mountRefreshRow struct {
	MountPoint       string                      `json:"mount_point"`
	RequestedTargets []string                    `json:"requested_targets,omitempty"`
	EffectiveTargets []string                    `json:"effective_targets,omitempty"`
	RefreshedTargets []string                    `json:"refreshed_targets,omitempty"`
	RefusedTargets   []string                    `json:"refused_targets,omitempty"`
	RuntimeStatus    contract.MountRuntimeStatus `json:"runtime_status,omitempty"`
}

func specMounts() engine.CommandSpec {
	return engine.CommandSpec{
		Name:    CommandMounts,
		Summary: "show mount contract status and run explicit scoped refresh actions",
		Manual:  "mounts [--fmt text|json]\nmounts refresh [--require-narrow] MOUNT_POINT...",
		Tips: []string{
			"mounts shows the active virtual mount contract surface visible to the runtime.",
			"mounts refresh is explicit and target-scoped; ordinary reads still do not trigger hidden refresh.",
			"Use --fmt json for machine-readable mount point/profile/status or refresh data.",
		},
		DefaultOutput:    "mount contract status rows or refresh result rows",
		StructuredOutput: "mount contract records or refresh records",
		StructuredFlags:  []string{"--fmt json"},
		StdinMode:        contract.BuiltinStdinNone,
		Examples: []string{
			"mounts",
			"mounts --fmt json",
			"mounts refresh " + "/" + "knowledge_base",
			"mounts refresh --require-narrow " + "/" + "knowledge_base" + "/" + "guide.md",
		},
		DetailedManual: LoadEmbeddedManual("mounts"),
		Run:            runMounts,
	}
}

func runMounts(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, out, code, ok := parseMountsArgs(args)
	if !ok {
		return out, code
	}
	switch opts.action {
	case "status":
		return runMountsStatus(runtime, opts)
	case "refresh":
		return runMountsRefresh(runtime, opts)
	default:
		return fmt.Sprintf("mounts: unsupported action %s", opts.action), contract.ExitCodeUsage
	}
}

func parseMountsArgs(args []string) (mountsOptions, string, int, bool) {
	opts := mountsOptions{action: "status", format: mountsFormatText}
	for idx := 0; idx < len(args); idx++ {
		arg := strings.TrimSpace(args[idx])
		switch {
		case arg == "":
			continue
		case idx == 0 && arg == "refresh":
			opts.action = "refresh"
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return opts, "mounts: --fmt requires one value: text|json", contract.ExitCodeUsage, false
			}
			idx++
			parsed, ok := parseMountsFormat(args[idx])
			if !ok {
				return opts, fmt.Sprintf("mounts: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage, false
			}
			opts.format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseMountsFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return opts, fmt.Sprintf("mounts: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage, false
			}
			opts.format = parsed
		case arg == "--require-narrow":
			opts.requireNarrow = true
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Sprintf("mounts: unsupported flag %s", arg), contract.ExitCodeUsage, false
		default:
			if opts.action != "refresh" {
				return opts, fmt.Sprintf("mounts: unsupported flag %s", arg), contract.ExitCodeUsage, false
			}
			opts.targets = append(opts.targets, normalizeMountRefreshTarget(arg))
		}
	}
	if opts.action == "refresh" && len(opts.targets) == 0 {
		return opts, "mounts refresh: expected one or more mount paths", contract.ExitCodeUsage, false
	}
	return opts, "", 0, true
}

func runMountsStatus(runtime engine.CommandRuntime, opts mountsOptions) (string, int) {
	rows := collectMountStatusRows(runtime)
	if opts.format == mountsFormatJSON {
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

func runMountsRefresh(runtime engine.CommandRuntime, opts mountsOptions) (string, int) {
	grouped, err := groupRefreshTargetsByMount(runtime.Ops.VirtualMounts, opts.targets)
	if err != nil {
		return fmt.Sprintf("mounts refresh: %v", err), contract.ExitCodeGeneral
	}
	rows := make([]mountRefreshRow, 0, len(grouped))
	for _, group := range grouped {
		result, err := contract.RefreshMount(runtime.Ctx, group.mount, contract.RefreshRequest{
			Targets:       append([]string(nil), group.targets...),
			RequireNarrow: opts.requireNarrow,
		})
		if err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return fmt.Sprintf("mounts refresh: %v", err), contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("mounts refresh: %v", err), contract.ExitCodeGeneral
		}
		row := mountRefreshRow{
			MountPoint:       group.mount.MountPoint(),
			RequestedTargets: append([]string(nil), group.targets...),
			EffectiveTargets: append([]string(nil), result.EffectiveTargets...),
			RefreshedTargets: append([]string(nil), result.RefreshedTargets...),
			RefusedTargets:   append([]string(nil), result.RefusedTargets...),
			RuntimeStatus:    readMountRuntimeStatus(runtime, group.mount),
		}
		sort.Strings(row.EffectiveTargets)
		sort.Strings(row.RefreshedTargets)
		sort.Strings(row.RefusedTargets)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].MountPoint < rows[j].MountPoint })
	if opts.format == mountsFormatJSON {
		raw, err := json.MarshalIndent(struct {
			Refresh []mountRefreshRow `json:"refresh"`
		}{Refresh: rows}, "", "  ")
		if err != nil {
			return fmt.Sprintf("mounts refresh: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		line := fmt.Sprintf("%s refreshed=%d refused=%d effective=%d", row.MountPoint, len(row.RefreshedTargets), len(row.RefusedTargets), len(row.EffectiveTargets))
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

func normalizeMountRefreshTarget(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

var errMountNotFound = errors.New("mount not found")

type refreshGroup struct {
	mount   contract.VirtualMount
	targets []string
}

func findMountForTarget(mounts []contract.VirtualMount, target string) (contract.VirtualMount, error) {
	target = normalizeMountRefreshTarget(target)
	var (
		best     contract.VirtualMount
		bestSize int
	)
	for _, mount := range mounts {
		if mount == nil {
			continue
		}
		point := normalizeMountRefreshTarget(mount.MountPoint())
		if target != point && !strings.HasPrefix(target, point+"/") {
			continue
		}
		if len(point) > bestSize {
			best = mount
			bestSize = len(point)
		}
	}
	if best == nil {
		return nil, errMountNotFound
	}
	return best, nil
}

func groupRefreshTargetsByMount(mounts []contract.VirtualMount, targets []string) ([]refreshGroup, error) {
	indexByPoint := make(map[string]int)
	groups := make([]refreshGroup, 0)
	for _, raw := range targets {
		target := normalizeMountRefreshTarget(raw)
		mount, err := findMountForTarget(mounts, target)
		if err != nil {
			return nil, fmt.Errorf("%s: mount not found", target)
		}
		point := normalizeMountRefreshTarget(mount.MountPoint())
		if idx, ok := indexByPoint[point]; ok {
			groups[idx].targets = append(groups[idx].targets, target)
			continue
		}
		indexByPoint[point] = len(groups)
		groups = append(groups, refreshGroup{mount: mount, targets: []string{target}})
	}
	return groups, nil
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

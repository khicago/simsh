package contract

import (
	"context"
	"strings"
)

type MountTruthModel string

const (
	MountTruthProjection MountTruthModel = "projection"
	MountTruthFactual    MountTruthModel = "factual"
)

type MountMaterializationMode string

const (
	MountMaterializationSnapshot MountMaterializationMode = "snapshot"
	MountMaterializationCached   MountMaterializationMode = "cached"
	MountMaterializationLive     MountMaterializationMode = "live"
)

type MountWriteSemantics string

const (
	MountWriteReadOnly MountWriteSemantics = "read_only"
	MountWriteStaged   MountWriteSemantics = "staged_write"
	MountWriteThrough  MountWriteSemantics = "write_through"
	MountWriteBack     MountWriteSemantics = "write_back"
)

type MountLatencyClass string

const (
	MountLatencyLocalFast      MountLatencyClass = "local_fast"
	MountLatencyLocalHeavy     MountLatencyClass = "local_heavy"
	MountLatencyRemoteModerate MountLatencyClass = "remote_moderate"
	MountLatencyRemoteHigh     MountLatencyClass = "remote_high_latency"
)

type MountCLIClass string

const (
	MountCLIList          MountCLIClass = "list"
	MountCLITree          MountCLIClass = "tree"
	MountCLIFind          MountCLIClass = "find"
	MountCLIRead          MountCLIClass = "read"
	MountCLIBulkRead      MountCLIClass = "bulk_read"
	MountCLIContentSearch MountCLIClass = "content_search"
	MountCLIMutate        MountCLIClass = "mutate"
)

type MountConsistency struct {
	PathReadAfterWrite bool `json:"path_read_after_write"`
	ListAfterWrite     bool `json:"list_after_write"`
	SearchAfterWrite   bool `json:"search_after_write"`
	RefreshRequired    bool `json:"refresh_required"`
}

type MountLatencySLO struct {
	P50MS int64 `json:"p50_ms,omitempty"`
	P95MS int64 `json:"p95_ms,omitempty"`
	P99MS int64 `json:"p99_ms,omitempty"`
}

type MountSLO struct {
	ListEntriesLatency    MountLatencySLO `json:"list_entries_latency,omitempty"`
	EnumeratePathsLatency MountLatencySLO `json:"enumerate_paths_latency,omitempty"`
	ReadManyLatency       MountLatencySLO `json:"read_many_latency,omitempty"`
	SearchLatency         MountLatencySLO `json:"search_latency,omitempty"`
	ApplyMutationsLatency MountLatencySLO `json:"apply_mutations_latency,omitempty"`
	MaxBatchCount         int             `json:"max_batch_count,omitempty"`
	MaxBatchBytes         int64           `json:"max_batch_bytes,omitempty"`
	MaxSearchPaths        int             `json:"max_search_paths,omitempty"`
	TimeoutSemantics      string          `json:"timeout_semantics,omitempty"`
	PartialResultMode     string          `json:"partial_result_mode,omitempty"`
	RetryabilityClass     string          `json:"retryability_class,omitempty"`
}

type MountProfile struct {
	TruthModel          MountTruthModel          `json:"truth_model,omitempty"`
	MaterializationMode MountMaterializationMode `json:"materialization_mode,omitempty"`
	WriteSemantics      MountWriteSemantics      `json:"write_semantics,omitempty"`
	LatencyClass        MountLatencyClass        `json:"latency_class,omitempty"`
	Consistency         MountConsistency         `json:"consistency,omitempty"`
	SupportedCLIClasses []MountCLIClass          `json:"supported_cli_classes,omitempty"`
	SLO                 MountSLO                 `json:"slo,omitempty"`
}

type MountEntry struct {
	Path string   `json:"path"`
	Name string   `json:"name"`
	Meta PathMeta `json:"meta"`
}

type ListEntriesRequest struct {
	Dir       string `json:"dir"`
	Recursive bool   `json:"recursive,omitempty"`
	MaxDepth  int    `json:"max_depth,omitempty"`
}

type ListEntriesResult struct {
	Entries []MountEntry `json:"entries"`
}

type EnumeratePathsRequest struct {
	Target    string `json:"target"`
	Recursive bool   `json:"recursive,omitempty"`
	MaxDepth  int    `json:"max_depth,omitempty"`
}

type EnumeratePathsResult struct {
	Entries []MountEntry `json:"entries"`
}

type MountContentEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ReadManyRequest struct {
	Paths      []string `json:"paths"`
	MaxEntries int      `json:"max_entries,omitempty"`
	MaxBytes   int64    `json:"max_bytes,omitempty"`
}

type ReadManyResult struct {
	Entries []MountContentEntry `json:"entries"`
}

type SearchCaseMode string

const (
	SearchCaseSensitive SearchCaseMode = "sensitive"
	SearchCaseIgnore    SearchCaseMode = "ignore"
	SearchCaseSmart     SearchCaseMode = "smart"
)

type SearchRequest struct {
	Pattern    string         `json:"pattern"`
	Regex      bool           `json:"regex,omitempty"`
	CaseMode   SearchCaseMode `json:"case_mode,omitempty"`
	Targets    []string       `json:"targets,omitempty"`
	Globs      []string       `json:"globs,omitempty"`
	Before     int            `json:"before,omitempty"`
	After      int            `json:"after,omitempty"`
	MaxResults int            `json:"max_results,omitempty"`
	ListFiles  bool           `json:"list_files,omitempty"`
}

type SearchRecord struct {
	Path string `json:"path,omitempty"`
	Line int    `json:"line"`
	Kind string `json:"kind"`
	Text string `json:"text,omitempty"`
}

type SearchResult struct {
	Records []SearchRecord `json:"records,omitempty"`
}

type MutationKind string

const (
	MutationWriteFile  MutationKind = "write_file"
	MutationAppend     MutationKind = "append_file"
	MutationEdit       MutationKind = "edit_file"
	MutationMakeDir    MutationKind = "make_dir"
	MutationRemoveFile MutationKind = "remove_file"
	MutationRemoveDir  MutationKind = "remove_dir"
)

type MutationSpec struct {
	Kind       MutationKind `json:"kind"`
	Path       string       `json:"path"`
	Content    string       `json:"content,omitempty"`
	OldString  string       `json:"old_string,omitempty"`
	NewString  string       `json:"new_string,omitempty"`
	ReplaceAll bool         `json:"replace_all,omitempty"`
}

type MutationBatch struct {
	Ops []MutationSpec `json:"ops"`
}

type MutationRecord struct {
	Kind         MutationKind `json:"kind"`
	Path         string       `json:"path"`
	Status       string       `json:"status,omitempty"`
	BytesWritten int          `json:"bytes_written,omitempty"`
}

type MutationResult struct {
	Records []MutationRecord `json:"records,omitempty"`
}

type RefreshRequest struct {
	Paths []string `json:"paths,omitempty"`
}

type RefreshResult struct {
	RefreshedPaths []string `json:"refreshed_paths,omitempty"`
}

type MountRuntimeStatus struct {
	Freshness       string `json:"freshness,omitempty"`
	Materialization string `json:"materialization,omitempty"`
	Detail          string `json:"detail,omitempty"`
	StatusError     string `json:"status_error,omitempty"`
}

type MountStats struct {
	RequestCounts map[string]int64 `json:"request_counts,omitempty"`
	ErrorCounts   map[string]int64 `json:"error_counts,omitempty"`
}

func NormalizeMountRuntimeStatus(status MountRuntimeStatus) MountRuntimeStatus {
	status.Freshness = strings.TrimSpace(status.Freshness)
	status.Materialization = strings.TrimSpace(status.Materialization)
	status.Detail = strings.TrimSpace(status.Detail)
	status.StatusError = strings.TrimSpace(status.StatusError)
	return status
}

// VirtualMount exposes a path subtree managed outside the primary filesystem.
// The base contract owns identity, profile, existence, and single-path metadata.
// Higher-fanout operations live in capability interfaces below.
type VirtualMount interface {
	MountPoint() string
	Profile() MountProfile
	Exists(ctx context.Context) (bool, error)
	StatPath(ctx context.Context, path string) (MountEntry, error)
	ReadContent(ctx context.Context, path string) (string, error)
}

type EntryLister interface {
	ListEntries(ctx context.Context, req ListEntriesRequest) (ListEntriesResult, error)
}

type PathEnumerator interface {
	EnumeratePaths(ctx context.Context, req EnumeratePathsRequest) (EnumeratePathsResult, error)
}

type BulkReader interface {
	ReadMany(ctx context.Context, req ReadManyRequest) (ReadManyResult, error)
}

type ContentSearcher interface {
	SearchContent(ctx context.Context, req SearchRequest) (SearchResult, error)
}

type Mutator interface {
	ApplyMutations(ctx context.Context, req MutationBatch) (MutationResult, error)
}

type Refresher interface {
	Refresh(ctx context.Context, req RefreshRequest) (RefreshResult, error)
}

type MountStatusProvider interface {
	MountStatus(ctx context.Context) (MountRuntimeStatus, error)
}

type StatsProvider interface {
	Stats(ctx context.Context) (MountStats, error)
}

type VirtualMountProvider interface {
	VirtualMounts() []VirtualMount
}

// BuiltinCatalog lets mounts query builtin command metadata without runtime coupling.
type BuiltinCatalog interface {
	BuiltinCommandDocs() []BuiltinCommandDoc
	LookupBuiltinDoc(name string) (BuiltinCommandDoc, bool)
}

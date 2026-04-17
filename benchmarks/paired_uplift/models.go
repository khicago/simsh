package main

import "time"

const (
	DefaultTaskManifestPath        = "benchmarks/paired_uplift/task_set.json"
	DefaultSnapshotPath            = "benchmarks/paired_uplift/reports/raw-baseline-20260406.json"
	DefaultArtifactPath            = "benchmarks/paired_uplift/reports/paired-baseline-20260406.json"
	DefaultSummaryPath             = "benchmarks/paired_uplift/reports/paired-baseline-20260406.md"
	DefaultFailureTaxonomyPath     = "benchmarks/paired_uplift/reports/paired-baseline-20260406.failures.json"
	pairedUpliftComparisonRule     = "same_agent_same_budget_only_substrate_changes"
	pairedProbeAgentID             = "paired_probe_agent_v1"
	substrateSimshFullSessioned    = "simsh_full_sessioned"
	substrateThinCoreStateless     = "thin_core_stateless"
	pairRunOrderAB                 = "AB"
	pairRunOrderBA                 = "BA"
	stepClassProgress              = "progress"
	stepClassRetry                 = "retry"
	stepClassWasted                = "wasted"
	stepClassEnvMisunderstanding   = "environment_misunderstanding"
	failureKindBudgetExhausted     = "budget_exhausted"
	failureKindBudgetAfterFallback = "budget_exhausted_after_fallback"
	failureKindUnexpectedOutput    = "unexpected_output"
	failureKindExecutionFailed     = "execution_failed"
	misunderstandingMissingJSON    = "missing_json_query_surface"
	misunderstandingMissingRG      = "missing_rg_front_door"
	misunderstandingNoSessionCWD   = "session_cwd_not_persistent"
	taxonomyBucketFailure          = "failure"
	taxonomyBucketMisunderstanding = "environment_misunderstanding"
	classificationSourceStructured = "structured_external_outcome"
	classificationSourceCompatText = "compatibility_output"
)

type TaskManifest struct {
	Version           int                  `json:"version"`
	ComparisonRule    string               `json:"comparison_rule"`
	AgentID           string               `json:"agent_id"`
	BaselineSubstrate string               `json:"baseline_substrate"`
	SimshSubstrate    string               `json:"simsh_substrate"`
	Tasks             []PairedTaskManifest `json:"tasks"`
	NonGoals          []string             `json:"non_goals,omitempty"`
}

type PairedTaskManifest struct {
	ScenarioID           string   `json:"scenario_id"`
	PairSeed             int64    `json:"pair_seed"`
	RunOrder             string   `json:"run_order"`
	MaxSteps             int      `json:"max_steps"`
	MaxObservationTokens int      `json:"max_observation_tokens"`
	ExpectedOutputs      []string `json:"expected_outputs,omitempty"`
	WhySelected          string   `json:"why_selected"`
}

type PairedTaskBudget struct {
	MaxSteps             int `json:"max_steps"`
	MaxObservationTokens int `json:"max_observation_tokens"`
}

type StepRecord struct {
	Index                    int                      `json:"index"`
	Label                    string                   `json:"label"`
	Command                  string                   `json:"command"`
	ExitCode                 int                      `json:"exit_code"`
	ObservationBytes         int                      `json:"observation_bytes"`
	ApproxObservationTokens  int                      `json:"approx_observation_tokens"`
	Classification           string                   `json:"classification"`
	ClassificationSource     string                   `json:"classification_source,omitempty"`
	Retry                    bool                     `json:"retry,omitempty"`
	Wasted                   bool                     `json:"wasted,omitempty"`
	EnvironmentMisunderstood bool                     `json:"environment_misunderstood,omitempty"`
	MisunderstandingKind     string                   `json:"misunderstanding_kind,omitempty"`
	Note                     string                   `json:"note,omitempty"`
	ExternalOutcomes         []ExternalOutcomeSummary `json:"external_outcomes,omitempty"`
}

type ExternalOutcomeSummary struct {
	Command      string `json:"command,omitempty"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	OutcomeKind  string `json:"outcome_kind,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
}

type SubstrateRunRecord struct {
	Substrate                    string       `json:"substrate"`
	Success                      bool         `json:"success"`
	FailureKind                  string       `json:"failure_kind,omitempty"`
	DurationMS                   int64        `json:"duration_ms"`
	Steps                        int          `json:"steps"`
	Retries                      int          `json:"retries"`
	WastedSteps                  int          `json:"wasted_steps"`
	EnvironmentMisunderstandings int          `json:"environment_misunderstandings"`
	ObservationBytes             int          `json:"observation_bytes"`
	ApproxObservationTokens      int          `json:"approx_observation_tokens"`
	WastedObservationTokens      int          `json:"wasted_observation_tokens"`
	LastMisunderstandingKind     string       `json:"last_misunderstanding_kind,omitempty"`
	StepsDetail                  []StepRecord `json:"steps_detail"`
	Notes                        []string     `json:"notes,omitempty"`
}

type PairRunRecord struct {
	ScenarioID      string             `json:"scenario_id"`
	Category        string             `json:"category"`
	TaskShape       string             `json:"task_shape"`
	Summary         string             `json:"summary"`
	TruthSurfaces   []string           `json:"truth_surfaces"`
	PairSeed        int64              `json:"pair_seed"`
	RunOrder        string             `json:"run_order"`
	AgentID         string             `json:"agent_id"`
	Budget          PairedTaskBudget   `json:"budget"`
	ExpectedOutputs []string           `json:"expected_outputs,omitempty"`
	WhySelected     string             `json:"why_selected"`
	EvidenceRefs    []string           `json:"evidence_refs"`
	Simsh           SubstrateRunRecord `json:"simsh"`
	Baseline        SubstrateRunRecord `json:"baseline"`
}

type PairedRunSnapshot struct {
	Version           int             `json:"version"`
	GeneratedAt       time.Time       `json:"generated_at"`
	ComparisonRule    string          `json:"comparison_rule"`
	TaskManifestPath  string          `json:"task_manifest_path"`
	AgentID           string          `json:"agent_id"`
	BaselineSubstrate string          `json:"baseline_substrate"`
	SimshSubstrate    string          `json:"simsh_substrate"`
	Tasks             []PairRunRecord `json:"tasks"`
}

type PairDelta struct {
	SuccessDelta                 int   `json:"success_delta"`
	RetryDelta                   int   `json:"retry_delta"`
	WastedStepDelta              int   `json:"wasted_step_delta"`
	WastedObservationTokensDelta int   `json:"wasted_observation_tokens_delta"`
	MisunderstandingDelta        int   `json:"misunderstanding_delta"`
	DurationMSDelta              int64 `json:"duration_ms_delta"`
}

type ComparedTask struct {
	ScenarioID      string             `json:"scenario_id"`
	Category        string             `json:"category"`
	TaskShape       string             `json:"task_shape"`
	Summary         string             `json:"summary"`
	TruthSurfaces   []string           `json:"truth_surfaces"`
	PairSeed        int64              `json:"pair_seed"`
	RunOrder        string             `json:"run_order"`
	AgentID         string             `json:"agent_id"`
	Budget          PairedTaskBudget   `json:"budget"`
	ExpectedOutputs []string           `json:"expected_outputs,omitempty"`
	WhySelected     string             `json:"why_selected"`
	EvidenceRefs    []string           `json:"evidence_refs"`
	Simsh           SubstrateRunRecord `json:"simsh"`
	Baseline        SubstrateRunRecord `json:"baseline"`
	Delta           PairDelta          `json:"delta"`
	Winner          string             `json:"winner"`
}

type AggregateSummary struct {
	TotalTasks                      int `json:"total_tasks"`
	SimshSuccessCount               int `json:"simsh_success_count"`
	BaselineSuccessCount            int `json:"baseline_success_count"`
	SimshRetries                    int `json:"simsh_retries"`
	BaselineRetries                 int `json:"baseline_retries"`
	SimshWastedSteps                int `json:"simsh_wasted_steps"`
	BaselineWastedSteps             int `json:"baseline_wasted_steps"`
	SimshMisunderstandings          int `json:"simsh_misunderstandings"`
	BaselineMisunderstandings       int `json:"baseline_misunderstandings"`
	SimshObservationTokens          int `json:"simsh_observation_tokens"`
	BaselineObservationTokens       int `json:"baseline_observation_tokens"`
	SimshWastedObservationTokens    int `json:"simsh_wasted_observation_tokens"`
	BaselineWastedObservationTokens int `json:"baseline_wasted_observation_tokens"`
}

type PairedUpliftArtifact struct {
	Version           int              `json:"version"`
	GeneratedAt       time.Time        `json:"generated_at"`
	ComparisonRule    string           `json:"comparison_rule"`
	TaskManifestPath  string           `json:"task_manifest_path"`
	AgentID           string           `json:"agent_id"`
	BaselineSubstrate string           `json:"baseline_substrate"`
	SimshSubstrate    string           `json:"simsh_substrate"`
	Summary           AggregateSummary `json:"summary"`
	Tasks             []ComparedTask   `json:"tasks"`
}

type FailureTaxonomyEntry struct {
	Bucket                string   `json:"bucket"`
	Runtime               string   `json:"runtime"`
	Kind                  string   `json:"kind"`
	Count                 int      `json:"count"`
	ScenarioIDs           []string `json:"scenario_ids"`
	ClassificationSources []string `json:"classification_sources,omitempty"`
	ExternalOutcomeKinds  []string `json:"external_outcome_kinds,omitempty"`
}

type FailureTaxonomyReport struct {
	Version            int                    `json:"version"`
	GeneratedAt        time.Time              `json:"generated_at"`
	SourceSnapshotPath string                 `json:"source_snapshot_path"`
	Entries            []FailureTaxonomyEntry `json:"entries"`
}

package externalmapping

import "time"

const (
	DefaultScenarioInventoryPath           = "benchmarks/external_mapping/scenario_inventory.json"
	DefaultTerminalBenchMappingPath        = "benchmarks/external_mapping/terminal_bench_mapping.json"
	DefaultSWEBenchLiveMappingPath         = "benchmarks/external_mapping/swe_bench_live_mapping.json"
	DefaultNativeBaselineReportPath        = "benchmarks/simsh_native_reference/reports/baseline-20260404.json"
	DefaultTerminalBenchPrototypeScopePath = "benchmarks/terminal_bench_compare/prototype_scope.json"
	DefaultTerminalBenchArtifactPath       = "benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.json"
	DefaultTerminalBenchSummaryPath        = "benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.md"
	terminalBenchExternalFamily            = "Terminal-Bench"
	ComparisonRoleDirectFit                = "direct_fit"
	ComparisonRoleTranslatedProof          = "translated_proof"
	PrototypeComparisonGoalDirectFit       = "direct_fit"
	PrototypeComparisonGoalTranslate       = "translation_proof"
	MappingStatusAsIs                      = "as_is"
	MappingStatusTranslated                = "translated"
	MappingStatusExcluded                  = "excluded"
)

type IdentityContract struct {
	CanonicalFields []string `json:"canonical_fields"`
	CuratedFields   []string `json:"curated_fields"`
}

type ScenarioInventoryRecord struct {
	ID            string   `json:"id"`
	Category      string   `json:"category"`
	TaskShape     string   `json:"task_shape"`
	Summary       string   `json:"summary"`
	TruthSurfaces []string `json:"truth_surfaces"`
}

type ScenarioInventory struct {
	Version          int                       `json:"version"`
	SourceBenchmark  string                    `json:"source_benchmark"`
	IdentityContract IdentityContract          `json:"identity_contract"`
	Scenarios        []ScenarioInventoryRecord `json:"scenarios"`
}

type FamilyMappingScenario struct {
	ScenarioID       string `json:"scenario_id"`
	Status           string `json:"status"`
	Rationale        string `json:"rationale"`
	TranslationNotes string `json:"translation_notes"`
}

type FamilyMapping struct {
	Version        int                     `json:"version"`
	ExternalFamily string                  `json:"external_family"`
	Relationship   string                  `json:"relationship"`
	Scenarios      []FamilyMappingScenario `json:"scenarios"`
}

type NativeScenarioReport struct {
	Name                  string   `json:"name"`
	Category              string   `json:"category"`
	Success               bool     `json:"success"`
	SessionScoped         bool     `json:"session_scoped"`
	AsyncCandidate        bool     `json:"async_candidate"`
	PatchWorkflow         bool     `json:"patch_workflow"`
	DurationMS            int64    `json:"duration_ms"`
	TraceChecksPassed     int      `json:"trace_checks_passed,omitempty"`
	TraceChecksTotal      int      `json:"trace_checks_total,omitempty"`
	TraceCompleteness     *float64 `json:"trace_completeness,omitempty"`
	AssertionChecksPassed int      `json:"assertion_checks_passed,omitempty"`
	AssertionChecksTotal  int      `json:"assertion_checks_total,omitempty"`
	AssertionCompleteness *float64 `json:"assertion_completeness,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

type NativeSuiteReport struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Scenarios   []NativeScenarioReport `json:"scenarios"`
}

type TerminalBenchPrototypeScope struct {
	Version        int                          `json:"version"`
	ExternalFamily string                       `json:"external_family"`
	ComparisonRule string                       `json:"comparison_rule"`
	Scenarios      []TerminalBenchPrototypeItem `json:"scenarios"`
	NonGoals       []string                     `json:"non_goals"`
}

type TerminalBenchPrototypeItem struct {
	ScenarioID           string   `json:"scenario_id"`
	Role                 string   `json:"role"`
	ExpectedStatus       string   `json:"expected_status"`
	ExternalTask         string   `json:"external_task"`
	ComparisonGoal       string   `json:"comparison_goal"`
	ComparableDimensions []string `json:"comparable_dimensions"`
	ExcludedDimensions   []string `json:"excluded_dimensions"`
	WhySelected          string   `json:"why_selected"`
}

type TerminalBenchComparisonSource struct {
	Benchmark   string    `json:"benchmark"`
	ReportPath  string    `json:"report_path"`
	GeneratedAt time.Time `json:"generated_at"`
}

type TerminalBenchNativeResult struct {
	Success               bool     `json:"success"`
	SessionScoped         bool     `json:"session_scoped"`
	AsyncCandidate        bool     `json:"async_candidate"`
	PatchWorkflow         bool     `json:"patch_workflow"`
	DurationMS            int64    `json:"duration_ms"`
	TraceCompleteness     *float64 `json:"trace_completeness,omitempty"`
	AssertionCompleteness *float64 `json:"assertion_completeness,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

type TerminalBenchComparedScenario struct {
	ScenarioID           string                    `json:"scenario_id"`
	Category             string                    `json:"category"`
	TaskShape            string                    `json:"task_shape"`
	Role                 string                    `json:"role"`
	MappingStatus        string                    `json:"mapping_status"`
	ComparisonGoal       string                    `json:"comparison_goal"`
	ExternalTask         string                    `json:"external_task"`
	Summary              string                    `json:"summary"`
	TruthSurfaces        []string                  `json:"truth_surfaces"`
	ComparableDimensions []string                  `json:"comparable_dimensions"`
	ExcludedDimensions   []string                  `json:"excluded_dimensions"`
	Rationale            string                    `json:"rationale"`
	TranslationNotes     string                    `json:"translation_notes,omitempty"`
	WhySelected          string                    `json:"why_selected"`
	EvidenceRefs         []string                  `json:"evidence_refs"`
	NativeResult         TerminalBenchNativeResult `json:"native_result"`
}

type TerminalBenchComparisonSummary struct {
	ComparedScenarios        int  `json:"compared_scenarios"`
	DirectFitScenarios       int  `json:"direct_fit_scenarios"`
	TranslatedProofScenarios int  `json:"translated_proof_scenarios"`
	AllNativeSuccessful      bool `json:"all_native_successful"`
}

type TerminalBenchComparisonArtifact struct {
	Version        int                             `json:"version"`
	ExternalFamily string                          `json:"external_family"`
	Source         TerminalBenchComparisonSource   `json:"source"`
	Scope          TerminalBenchPrototypeScope     `json:"scope"`
	Summary        TerminalBenchComparisonSummary  `json:"summary"`
	Scenarios      []TerminalBenchComparedScenario `json:"scenarios"`
}

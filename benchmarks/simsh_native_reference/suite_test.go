package main

import "testing"

func TestRunSuitePassesFirstPassGates(t *testing.T) {
	report, err := runSuite()
	if err != nil {
		t.Fatalf("runSuite failed: %v", err)
	}
	if len(report.Scenarios) < 5 {
		t.Fatalf("expected benchmark scenarios, got %+v", report.Scenarios)
	}
	for _, gate := range report.Gates {
		if !gate.Pass {
			t.Fatalf("expected gate %s to pass: %+v", gate.Name, report)
		}
	}
}

func TestComputeMetricsIgnoresNonTraceAssertions(t *testing.T) {
	metrics := computeMetrics([]ScenarioReport{
		{
			Name:              "trace-only",
			TraceChecksPassed: 2,
			TraceChecksTotal:  4,
		},
		{
			Name:                  "behavior-only",
			AssertionChecksPassed: 3,
			AssertionChecksTotal:  4,
		},
	})

	if metrics.TraceCompleteness != 0.5 {
		t.Fatalf("computeMetrics(...).TraceCompleteness = %v, want 0.5", metrics.TraceCompleteness)
	}
}

func TestScenarioSucceededRequiresCompleteEvidence(t *testing.T) {
	tests := []struct {
		name            string
		tracePassed     int
		traceTotal      int
		assertionPassed int
		assertionTotal  int
		want            bool
	}{
		{name: "all complete", tracePassed: 2, traceTotal: 2, assertionPassed: 3, assertionTotal: 3, want: true},
		{name: "trace incomplete", tracePassed: 1, traceTotal: 2, assertionPassed: 3, assertionTotal: 3, want: false},
		{name: "assertions incomplete", tracePassed: 2, traceTotal: 2, assertionPassed: 2, assertionTotal: 3, want: false},
		{name: "no trace checks", tracePassed: 0, traceTotal: 0, assertionPassed: 3, assertionTotal: 3, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scenarioSucceeded(tt.tracePassed, tt.traceTotal, tt.assertionPassed, tt.assertionTotal); got != tt.want {
				t.Fatalf("scenarioSucceeded(trace=%d/%d assertions=%d/%d) = %t, want %t", tt.tracePassed, tt.traceTotal, tt.assertionPassed, tt.assertionTotal, got, tt.want)
			}
		})
	}
}

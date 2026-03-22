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

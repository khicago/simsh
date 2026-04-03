package main

import (
	"flag"
	"fmt"
	"os"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
)

func main() {
	scopePath := flag.String("scope", externalmapping.DefaultTerminalBenchPrototypeScopePath, "prototype scope JSON")
	reportPath := flag.String("report", externalmapping.DefaultNativeBaselineReportPath, "native benchmark report JSON to compare")
	outJSON := flag.String("out-json", externalmapping.DefaultTerminalBenchArtifactPath, "optional path to write the comparison JSON artifact")
	outMD := flag.String("out-md", externalmapping.DefaultTerminalBenchSummaryPath, "optional path to write the markdown comparison report")
	flag.Parse()

	scope, err := externalmapping.LoadTerminalBenchPrototypeScope(*scopePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal-bench-compare: %v\n", err)
		os.Exit(1)
	}
	inventory, err := externalmapping.LoadScenarioInventory("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal-bench-compare: %v\n", err)
		os.Exit(1)
	}
	mapping, err := externalmapping.LoadFamilyMapping(externalmapping.DefaultTerminalBenchMappingPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal-bench-compare: %v\n", err)
		os.Exit(1)
	}
	source, err := externalmapping.LoadNativeSuiteReport(*reportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal-bench-compare: %v\n", err)
		os.Exit(1)
	}

	report, err := externalmapping.BuildTerminalBenchComparison(inventory, mapping, source, scope, *reportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal-bench-compare: %v\n", err)
		os.Exit(1)
	}

	rawJSON, err := externalmapping.MarshalTerminalBenchComparisonJSON(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal-bench-compare: marshal json failed: %v\n", err)
		os.Exit(1)
	}
	if *outJSON != "" {
		if err := os.WriteFile(*outJSON, rawJSON, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "terminal-bench-compare: write json failed: %v\n", err)
			os.Exit(1)
		}
	}
	if *outMD != "" {
		if err := os.WriteFile(*outMD, []byte(externalmapping.RenderTerminalBenchComparisonMarkdown(report)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "terminal-bench-compare: write markdown failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(string(rawJSON))
}

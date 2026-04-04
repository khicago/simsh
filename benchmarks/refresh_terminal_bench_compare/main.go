package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
)

func main() {
	rootDir := flag.String("root", ".", "repository root used to run the refresh commands")
	goBinary := flag.String("go", "go", "go binary to use for refresh commands")
	flag.Parse()

	plan := externalmapping.DefaultTerminalBenchRefreshPlan(*rootDir)
	plan.GoBinary = *goBinary
	if err := externalmapping.RunTerminalBenchRefresh(context.Background(), plan, externalmapping.ExecRunner{}); err != nil {
		fmt.Fprintf(os.Stderr, "refresh-terminal-bench-compare: %v\n", err)
		os.Exit(1)
	}
}

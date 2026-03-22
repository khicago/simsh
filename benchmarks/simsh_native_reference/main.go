package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	outPath := flag.String("out", "", "optional path to write the JSON report")
	flag.Parse()

	report, err := runSuite()
	if err != nil {
		fmt.Fprintf(os.Stderr, "simsh-native-reference: %v\n", err)
		os.Exit(1)
	}

	raw, err := marshalReport(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "simsh-native-reference: marshal report failed: %v\n", err)
		os.Exit(1)
	}

	if *outPath != "" {
		if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "simsh-native-reference: write report failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println(string(raw))
	for _, gate := range report.Gates {
		if !gate.Pass {
			os.Exit(2)
		}
	}
}

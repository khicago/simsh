package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	taskSetPath := flag.String("task-set", DefaultTaskManifestPath, "paired uplift task manifest JSON")
	outSnapshot := flag.String("out-snapshot", DefaultSnapshotPath, "optional path to write the raw paired run snapshot")
	outJSON := flag.String("out-json", DefaultArtifactPath, "optional path to write the aggregate uplift JSON artifact")
	outMD := flag.String("out-md", DefaultSummaryPath, "optional path to write the markdown uplift summary")
	outFailures := flag.String("out-failures", DefaultFailureTaxonomyPath, "optional path to write the failure taxonomy JSON artifact")
	flag.Parse()

	manifest, err := LoadTaskManifest(*taskSetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paired-uplift: %v\n", err)
		os.Exit(1)
	}
	snapshot, err := RunPairedUplift(context.Background(), manifest, *taskSetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paired-uplift: %v\n", err)
		os.Exit(1)
	}
	artifact, err := BuildPairedUpliftArtifact(snapshot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paired-uplift: %v\n", err)
		os.Exit(1)
	}
	taxonomy := BuildFailureTaxonomy(snapshot, *outSnapshot)

	snapshotJSON, err := MarshalPairedRunSnapshotJSON(snapshot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paired-uplift: marshal snapshot failed: %v\n", err)
		os.Exit(1)
	}
	artifactJSON, err := MarshalPairedUpliftArtifactJSON(artifact)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paired-uplift: marshal artifact failed: %v\n", err)
		os.Exit(1)
	}
	taxonomyJSON, err := MarshalFailureTaxonomyJSON(taxonomy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paired-uplift: marshal taxonomy failed: %v\n", err)
		os.Exit(1)
	}
	if *outSnapshot != "" {
		if err := writeOutputFile(*outSnapshot, snapshotJSON); err != nil {
			fmt.Fprintf(os.Stderr, "paired-uplift: write snapshot failed: %v\n", err)
			os.Exit(1)
		}
	}
	if *outJSON != "" {
		if err := writeOutputFile(*outJSON, artifactJSON); err != nil {
			fmt.Fprintf(os.Stderr, "paired-uplift: write artifact failed: %v\n", err)
			os.Exit(1)
		}
	}
	if *outMD != "" {
		if err := writeOutputFile(*outMD, []byte(RenderPairedUpliftMarkdown(artifact, taxonomy))); err != nil {
			fmt.Fprintf(os.Stderr, "paired-uplift: write markdown failed: %v\n", err)
			os.Exit(1)
		}
	}
	if *outFailures != "" {
		if err := writeOutputFile(*outFailures, taxonomyJSON); err != nil {
			fmt.Fprintf(os.Stderr, "paired-uplift: write failure taxonomy failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(string(artifactJSON))
}

func writeOutputFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

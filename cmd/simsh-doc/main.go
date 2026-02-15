package main

import (
	"fmt"
	"os"
	"path/filepath"

	runtimecmd "github.com/khicago/simsh/pkg/cmd"
)

func main() {
	outPath := "simsh.md"
	if len(os.Args) > 1 && os.Args[1] != "" {
		outPath = os.Args[1]
	}
	content := runtimecmd.DescribeMarkdown()
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s failed: %v\n", outPath, err)
		os.Exit(1)
	}
	abs, _ := filepath.Abs(outPath)
	fmt.Printf("generated %s\n", abs)
}

package diff

import (
	"embed"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/builtin/shared"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"diff /knowledge_base/v1.md /knowledge_base/v2.md"}

//go:embed manual.md
var manualFS embed.FS

func detailedManual() string {
	data, err := manualFS.ReadFile("manual.md")
	if err != nil {
		return ""
	}
	return string(data)
}

func Spec() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   "diff",
		Manual: "diff PATH1 PATH2",
		Tips: []string{
			"Compares two files line by line.",
			"Exit code 0 if identical, 1 if different.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) != 2 {
		return "diff: expected exactly two file paths", contract.ExitCodeUsage
	}
	path1, err := runtime.Ops.RequireAbsolutePath(args[0])
	if err != nil {
		return fmt.Sprintf("diff: %v", err), contract.ExitCodeUsage
	}
	path2, err := runtime.Ops.RequireAbsolutePath(args[1])
	if err != nil {
		return fmt.Sprintf("diff: %v", err), contract.ExitCodeUsage
	}
	raw1, err := runtime.Ops.ReadRawContent(runtime.Ctx, path1)
	if err != nil {
		return fmt.Sprintf("diff: %v", err), contract.ExitCodeGeneral
	}
	raw2, err := runtime.Ops.ReadRawContent(runtime.Ctx, path2)
	if err != nil {
		return fmt.Sprintf("diff: %v", err), contract.ExitCodeGeneral
	}
	if raw1 == raw2 {
		return "", 0
	}
	lines1 := shared.SplitRawLines(raw1)
	lines2 := shared.SplitRawLines(raw2)
	out := simpleDiff(path1, path2, lines1, lines2)
	return out, contract.ExitCodeGeneral
}

func simpleDiff(path1, path2 string, lines1, lines2 []string) string {
	lcs := lcsTable(lines1, lines2)
	var out []string
	out = append(out, fmt.Sprintf("--- %s", path1))
	out = append(out, fmt.Sprintf("+++ %s", path2))

	i, j := len(lines1), len(lines2)
	var changes []string
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && lines1[i-1] == lines2[j-1] {
			changes = append(changes, " "+lines1[i-1])
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			changes = append(changes, "+"+lines2[j-1])
			j--
		} else {
			changes = append(changes, "-"+lines1[i-1])
			i--
		}
	}
	for k := len(changes) - 1; k >= 0; k-- {
		out = append(out, changes[k])
	}
	return strings.Join(out, "\n")
}

func lcsTable(a, b []string) [][]int {
	m, n := len(a), len(b)
	table := make([][]int, m+1)
	for i := range table {
		table[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				table[i][j] = table[i-1][j-1] + 1
			} else if table[i-1][j] >= table[i][j-1] {
				table[i][j] = table[i-1][j]
			} else {
				table[i][j] = table[i][j-1]
			}
		}
	}
	return table
}

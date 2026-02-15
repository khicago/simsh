package mount

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

const virtualTestDir = "/test"

//go:embed testdata/baseline/*/cases.json
var baselineCorpusFS embed.FS

type baselineCorpusCase struct {
	Name     string `json:"name"`
	Script   string `json:"script"`
	Stdout   string `json:"stdout"`
	ExitCode int    `json:"exit_code"`
}

func NewBaselineCorpusMount() (contract.VirtualMount, error) {
	mount, err := buildBaselineCorpusMount()
	if err != nil {
		return nil, err
	}
	return mount, nil
}

func buildBaselineCorpusMount() (contract.VirtualMount, error) {
	matches, err := fs.Glob(baselineCorpusFS, "testdata/baseline/*/cases.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return nil, fmt.Errorf("baseline corpus is empty")
	}

	files := map[string]string{
		virtualTestDir + "/README.md": strings.Join([]string{
			"# simsh baseline corpus",
			"",
			"This mount is generated from embedded baseline cases.",
			"Profiles are available under /test/<profile>/.",
		}, "\n") + "\n",
	}
	for _, match := range matches {
		raw, readErr := baselineCorpusFS.ReadFile(match)
		if readErr != nil {
			return nil, readErr
		}
		profile := baselineProfileFromPath(match)
		if strings.TrimSpace(profile) == "" {
			return nil, fmt.Errorf("invalid baseline profile path: %s", match)
		}
		files[virtualTestDir+"/"+profile+"/cases.json"] = string(raw)
		files[virtualTestDir+"/"+profile+"/README.md"] = fmt.Sprintf(
			"# baseline profile: %s\n\n- Source: %s\n- Cases are mirrored under ./cases/*.sh\n",
			profile,
			match,
		)

		cases := make([]baselineCorpusCase, 0)
		if err := json.Unmarshal(raw, &cases); err != nil {
			return nil, fmt.Errorf("parse baseline profile %s failed: %w", profile, err)
		}
		for _, c := range cases {
			caseName := sanitizeMountEntryName(c.Name)
			if caseName == "" {
				continue
			}
			script := c.Script
			if script != "" && !strings.HasSuffix(script, "\n") {
				script += "\n"
			}
			files[fmt.Sprintf("%s/%s/cases/%s.sh", virtualTestDir, profile, caseName)] = script
			files[fmt.Sprintf("%s/%s/expect/%s.txt", virtualTestDir, profile, caseName)] = fmt.Sprintf(
				"exit_code=%d\nstdout=\n%s",
				c.ExitCode,
				c.Stdout,
			)
		}
	}

	return NewStaticMount(virtualTestDir, "test", files)
}

func baselineProfileFromPath(pathValue string) string {
	parts := strings.Split(path.Clean(pathValue), "/")
	if len(parts) < 4 {
		return ""
	}
	if parts[0] != "testdata" || parts[1] != "baseline" {
		return ""
	}
	return strings.TrimSpace(parts[2])
}

func sanitizeMountEntryName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		" ", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(trimmed)
}

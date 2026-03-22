package builtin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

type mutationPathStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

func extractMutationOutputFlags(commandName string, args []string) (filtered []string, confirm bool, jsonOutput bool, out string, code int, ok bool) {
	filtered = make([]string, 0, len(args))
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch arg {
		case "--confirm":
			confirm = true
			continue
		case "--json":
			jsonOutput = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, false, false, fmt.Sprintf("%s: unsupported flag %s", commandName, arg), contract.ExitCodeUsage, false
		}
		filtered = append(filtered, arg)
	}
	return filtered, confirm, jsonOutput, "", 0, true
}

func renderPathStatusMutation(confirm bool, jsonOutput bool, entries []mutationPathStatus) (string, int, error) {
	if jsonOutput {
		raw, err := json.Marshal(struct {
			Entries []mutationPathStatus `json:"entries"`
		}{
			Entries: entries,
		})
		if err != nil {
			return "", 0, err
		}
		return string(raw), 0, nil
	}
	if confirm {
		lines := make([]string, 0, len(entries))
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("%s %s", entry.Status, entry.Path))
		}
		return strings.Join(lines, "\n"), 0, nil
	}
	return "", 0, nil
}

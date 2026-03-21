package builtin

import (
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

func currentWorkingDir(ops contract.Ops) string {
	if ops.GetWorkingDir != nil {
		if cwd := strings.TrimSpace(ops.GetWorkingDir()); cwd != "" {
			return cwd
		}
	}
	if cwd := strings.TrimSpace(ops.WorkingDir); cwd != "" {
		return cwd
	}
	if root := strings.TrimSpace(ops.RootDir); root != "" {
		return root
	}
	return "/"
}

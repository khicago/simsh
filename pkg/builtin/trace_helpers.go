package builtin

import (
	"strings"

	"github.com/khicago/simsh/pkg/engine"
)

func traceDeniedPaths(runtime engine.CommandRuntime, paths ...string) {
	for _, pathValue := range paths {
		trimmed := strings.TrimSpace(pathValue)
		if trimmed == "" {
			continue
		}
		engine.MarkTraceRequestedPath(runtime.Ctx, trimmed)
		engine.MarkTraceDeniedPath(runtime.Ctx, trimmed)
	}
}

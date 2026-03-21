package builtin

import (
	"errors"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type pathCheck struct {
	path               string
	op                 contract.PathOp
	unsupportedMessage string
}

func preflightPathChecks(runtime engine.CommandRuntime, command string, checks []pathCheck) (string, int, bool) {
	for _, check := range checks {
		if err := runtime.Ops.CheckPathOp(runtime.Ctx, check.op, check.path); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				message := check.unsupportedMessage
				if message == "" {
					message = fmt.Sprintf("%s: not supported", command)
				}
				return message, contract.ExitCodeUnsupported, false
			}
			return fmt.Sprintf("%s: %v", command, err), contract.ExitCodeGeneral, false
		}
	}
	return "", 0, true
}

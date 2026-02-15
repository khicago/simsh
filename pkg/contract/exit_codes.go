package contract

import "errors"

const (
	ExitCodeUsage       = 2
	ExitCodeGeneral     = 1
	ExitCodeUnsupported = 127
)

var ErrUnsupported = errors.New("simsh: unsupported operation")

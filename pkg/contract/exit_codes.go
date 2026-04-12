package contract

import "errors"

const (
	ExitCodeUsage       = 2
	ExitCodeGeneral     = 1
	ExitCodeUnsupported = 127
)

var ErrUnsupported = errors.New("simsh: unsupported operation")

var ErrExternalCommandNotFound = errors.New("simsh: external command not found")

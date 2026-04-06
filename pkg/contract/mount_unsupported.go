package contract

import (
	"errors"
)

type MountUnsupportedError struct {
	MountPoint   string
	Capability   string
	LatencyClass MountLatencyClass
	Detail       string
}

func (e *MountUnsupportedError) Error() string {
	if e == nil || e.Detail == "" {
		return ErrUnsupported.Error()
	}
	return ErrUnsupported.Error() + ": " + e.Detail
}

func (e *MountUnsupportedError) Unwrap() error {
	return ErrUnsupported
}

func AllowsUnsupportedFallback(err error) bool {
	if !errors.Is(err, ErrUnsupported) {
		return false
	}
	var mountErr *MountUnsupportedError
	if errors.As(err, &mountErr) && mountErr.LatencyClass == MountLatencyRemoteHigh {
		return false
	}
	return true
}

func IsRemoteHighLatencyUnsupported(err error) bool {
	var mountErr *MountUnsupportedError
	return errors.As(err, &mountErr) && mountErr.LatencyClass == MountLatencyRemoteHigh
}

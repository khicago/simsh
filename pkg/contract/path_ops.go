package contract

// PathOp is a coarse-grained operation type used by higher-level commands to
// preflight path access before performing multi-step mutations (e.g. mv).
//
// The underlying filesystem adapters still enforce their own rules; this hook
// exists so filesystems and virtual overlays can provide consistent, early
// rejection and avoid partial side effects.
type PathOp string

const (
	PathOpRead   PathOp = "read"
	PathOpWrite  PathOp = "write"
	PathOpMkdir  PathOp = "mkdir"
	PathOpRemove PathOp = "remove"
)

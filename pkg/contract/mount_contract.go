package contract

import "context"

// VirtualMount exposes a path subtree managed outside the primary filesystem.
type VirtualMount interface {
	MountPoint() string
	Exists(ctx context.Context) (bool, error)
	ListChildren(ctx context.Context, dir string) ([]string, error)
	IsDirPath(ctx context.Context, path string) (bool, error)
	ReadRawContent(ctx context.Context, path string) (string, error)
	CollectFilesUnder(ctx context.Context, target string) ([]string, error)
	ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error)
	DescribePath(ctx context.Context, path string) (PathMeta, error)
}

type VirtualMountProvider interface {
	VirtualMounts() []VirtualMount
}

// BuiltinCatalog lets mounts query builtin command metadata without runtime coupling.
type BuiltinCatalog interface {
	BuiltinCommandDocs() []BuiltinCommandDoc
	LookupBuiltinDoc(name string) (BuiltinCommandDoc, bool)
}

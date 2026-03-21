package contract

import (
	"context"
	"strings"
)

type PathRootProvider interface {
	RootDir() string
}

type PathResolver interface {
	RequireAbsolutePath(raw string) (string, error)
}

type DirectoryReader interface {
	ListChildren(ctx context.Context, dir string) ([]string, error)
	IsDirPath(ctx context.Context, path string) (bool, error)
}

type ContentReader interface {
	ReadRawContent(ctx context.Context, path string) (string, error)
}

type SearchResolver interface {
	ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error)
	CollectFilesUnder(ctx context.Context, target string) ([]string, error)
}

type ContentWriter interface {
	WriteFile(ctx context.Context, filePath string, content string) error
	AppendFile(ctx context.Context, filePath string, content string) error
	EditFile(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error
}

type DirectoryCreator interface {
	MakeDir(ctx context.Context, dirPath string) error
}

type FileRemover interface {
	RemoveFile(ctx context.Context, filePath string) error
}

type DirectoryRemover interface {
	RemoveDir(ctx context.Context, dirPath string) error
}

type PathDescriber interface {
	DescribePath(ctx context.Context, path string) (PathMeta, error)
}

type ExternalCommandLister interface {
	ListExternalCommands(ctx context.Context) ([]ExternalCommand, error)
}

type ExternalCommandRunner interface {
	RunExternalCommand(ctx context.Context, req ExternalCommandRequest) (ExternalCommandResult, error)
}

type ExternalManualReader interface {
	ReadExternalManual(ctx context.Context, command string) (string, error)
}

type PathEnvProvider interface {
	PathEnv() []string
}

type PathOpChecker interface {
	CheckPathOp(ctx context.Context, op PathOp, path string) error
}

// Filesystem is the stable integration contract for embedding simsh.
type Filesystem interface {
	PathRootProvider
	PathResolver
	DirectoryReader
	ContentReader
	SearchResolver
	ContentWriter
}

// Ops is callback wiring used by the engine and adapters.
type Ops struct {
	RootDir             string
	WorkingDir          string
	GetWorkingDir       func() string
	ChangeWorkingDir    func(ctx context.Context, raw string) (string, error)
	ResolvePath         func(raw string) (string, error)
	RequireAbsolutePath func(raw string) (string, error)
	ListChildren        func(ctx context.Context, dir string) ([]string, error)
	IsDirPath           func(ctx context.Context, path string) (bool, error)
	DescribePath        func(ctx context.Context, path string) (PathMeta, error)
	FormatLSLongRow     LSLongRowFormatter
	ReadRawContent      func(ctx context.Context, path string) (string, error)
	ResolveSearchPaths  func(ctx context.Context, target string, recursive bool) ([]string, error)
	CollectFilesUnder   func(ctx context.Context, target string) ([]string, error)
	WriteFile           func(ctx context.Context, filePath string, content string) error
	AppendFile          func(ctx context.Context, filePath string, content string) error
	EditFile            func(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error
	MakeDir             func(ctx context.Context, dirPath string) error
	RemoveFile          func(ctx context.Context, filePath string) error
	RemoveDir           func(ctx context.Context, dirPath string) error
	// CheckPathOp is an optional preflight hook for commands to validate path
	// access before executing multi-step mutations. Virtual overlays may use it
	// to provide consistent "unsupported" errors and avoid partial writes.
	CheckPathOp          func(ctx context.Context, op PathOp, path string) error
	ListExternalCommands func(ctx context.Context) ([]ExternalCommand, error)
	RunExternalCommand   func(ctx context.Context, req ExternalCommandRequest) (ExternalCommandResult, error)
	ReadExternalManual   func(ctx context.Context, command string) (string, error)
	CommandAliases       map[string][]string
	EnvVars              map[string]string
	RCFiles              []string
	PathEnv              []string
	VirtualMounts        []VirtualMount
	Profile              CompatibilityProfile
	Policy               ExecutionPolicy
	AuditSink            AuditSink
}

func OpsFromFilesystem(fs Filesystem) Ops {
	root := strings.TrimSpace(fs.RootDir())
	if root == "" {
		root = "/"
	}
	ops := Ops{
		WorkingDir:          root,
		RootDir:             root,
		RequireAbsolutePath: fs.RequireAbsolutePath,
		ListChildren:        fs.ListChildren,
		IsDirPath:           fs.IsDirPath,
		ReadRawContent:      fs.ReadRawContent,
		ResolveSearchPaths:  fs.ResolveSearchPaths,
		CollectFilesUnder:   fs.CollectFilesUnder,
		WriteFile:           fs.WriteFile,
		AppendFile:          fs.AppendFile,
		EditFile:            fs.EditFile,
		Profile:             DefaultProfile(),
		Policy:              DefaultPolicy(),
	}
	if describer, ok := fs.(PathDescriber); ok {
		ops.DescribePath = describer.DescribePath
	}
	if lister, ok := fs.(ExternalCommandLister); ok {
		ops.ListExternalCommands = lister.ListExternalCommands
	}
	if runner, ok := fs.(ExternalCommandRunner); ok {
		ops.RunExternalCommand = runner.RunExternalCommand
	}
	if manual, ok := fs.(ExternalManualReader); ok {
		ops.ReadExternalManual = manual.ReadExternalManual
	}
	if dc, ok := fs.(DirectoryCreator); ok {
		ops.MakeDir = dc.MakeDir
	}
	if fr, ok := fs.(FileRemover); ok {
		ops.RemoveFile = fr.RemoveFile
	}
	if dr, ok := fs.(DirectoryRemover); ok {
		ops.RemoveDir = dr.RemoveDir
	}
	if provider, ok := fs.(VirtualMountProvider); ok {
		ops.VirtualMounts = append(ops.VirtualMounts, provider.VirtualMounts()...)
	}
	if pathEnv, ok := fs.(PathEnvProvider); ok {
		ops.PathEnv = append(ops.PathEnv, pathEnv.PathEnv()...)
	}
	if checker, ok := fs.(PathOpChecker); ok {
		ops.CheckPathOp = checker.CheckPathOp
	}
	return ops
}

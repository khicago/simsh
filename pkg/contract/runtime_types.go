package contract

import "context"

const (
	BuiltinStdinNone     = "none"
	BuiltinStdinOptional = "optional"
	BuiltinStdinRequired = "required"

	BuiltinMutationReadOnly = "read_only"
	BuiltinMutationMutates  = "mutates"

	BuiltinSuccessContent = "content"
	BuiltinSuccessSilent  = "silent"
	BuiltinSuccessSummary = "summary"

	BuiltinPipeWeak   = "weak"
	BuiltinPipeGood   = "good"
	BuiltinPipeStrong = "strong"
)

// PathMeta carries semantic metadata for long-listing rendering.
type PathMeta struct {
	Exists           bool
	IsDir            bool
	Kind             string
	Access           string
	Capabilities     []string
	LineCount        int
	FrontMatterLines int
	SpeakerRows      int
	UserRelevance    string
}

// LSLongRow is the normalized row data for ls -l rendering.
type LSLongRow struct {
	DisplayPath      string
	Path             string
	Exists           bool
	IsDir            bool
	Kind             string
	Access           string
	Capabilities     []string
	LineCount        int
	FrontMatterLines int
	SpeakerRows      int
	UserRelevance    string
}

type LSLongRowFormatter func(ctx context.Context, row LSLongRow) (formatted string, handled bool)

// ExternalCommand describes one business command exposed under /bin.
type ExternalCommand struct {
	Name    string
	Summary string
}

// ExternalCommandRequest carries one external command invocation.
type ExternalCommandRequest struct {
	Command  string
	RawPath  string
	Args     []string
	Stdin    string
	HasStdin bool
}

// ExternalCommandResult is the output of one external command invocation.
type ExternalCommandResult struct {
	Stdout           string
	Stderr           string
	ExitCode         int
	RawExitCode      *int
	CanonicalTarget  string
	ProviderError    string
	TerminationKind  string
}

// BuiltinCommandDoc provides prompt/introspection-friendly command metadata.
type BuiltinCommandDoc struct {
	Name             string
	Summary          string
	Manual           string
	Tips             []string
	Examples         []string
	DetailedManual   string
	Capabilities     []string
	StdinMode        string
	Operands         string
	DefaultOutput    string
	StructuredOutput string
	StructuredFlags  []string
	MutationKind     string
	SuccessOutput    string
	PipeBehavior     string
	ExitCodes        []string
}

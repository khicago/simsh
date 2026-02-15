package engine

type redirectKind int

const (
	redirectInputFile redirectKind = iota
	redirectInputHereDoc
	redirectOutputWrite
	redirectOutputAppend
)

type commandRedirect struct {
	kind        redirectKind
	target      string
	hereDocBody string
}

type parsedCommand struct {
	args   []string
	redirs []commandRedirect
}

type parsedPipeline struct {
	commands []parsedCommand
}

type conditionalPipeline struct {
	op       string
	pipeline parsedPipeline
}

type parsedStatement struct {
	pipelines []conditionalPipeline
}

type scriptTokenKind int

const (
	scriptTokenWord scriptTokenKind = iota
	scriptTokenPipe
	scriptTokenAnd
	scriptTokenOr
	scriptTokenSemi
	scriptTokenRedirIn
	scriptTokenRedirOut
	scriptTokenRedirAppend
	scriptTokenRedirHereDoc
)

type scriptToken struct {
	kind  scriptTokenKind
	value string
}

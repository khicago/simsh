package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
)

const maxExecuteStatements = 64

func (e *Engine) runScript(ctx context.Context, cmdline string, ops contract.Ops) (string, int) {
	statements, err := parseScript(cmdline)
	if err != nil {
		return fmt.Sprintf("execute: %v", err), contract.ExitCodeUsage
	}
	if len(statements) == 0 {
		return "", 0
	}
	return e.executeStatements(ctx, statements, ops)
}

func (e *Engine) executeStatements(ctx context.Context, statements []parsedStatement, ops contract.Ops) (string, int) {
	if len(statements) > maxExecuteStatements {
		return fmt.Sprintf("execute: statement count exceeds limit (%d)", maxExecuteStatements), contract.ExitCodeGeneral
	}
	lastOutput := ""
	lastCode := 0
	executed := false

	for _, statement := range statements {
		statementOutput := ""
		statementCode := 0
		statementExecuted := false

		for idx, node := range statement.pipelines {
			shouldRun := true
			if idx > 0 {
				switch node.op {
				case "&&":
					shouldRun = statementCode == 0
				case "||":
					shouldRun = statementCode != 0
				}
			}
			if !shouldRun {
				continue
			}

			out, code := e.executePipeline(ctx, node.pipeline, ops)
			statementOutput = out
			statementCode = code
			statementExecuted = true
		}

		if statementExecuted {
			lastOutput = statementOutput
			lastCode = statementCode
			executed = true
		}
	}

	if !executed {
		return "", 0
	}
	return lastOutput, lastCode
}

func (e *Engine) executePipeline(ctx context.Context, pipeline parsedPipeline, ops contract.Ops) (string, int) {
	if ops.Policy.MaxPipelineDepth > 0 && len(pipeline.commands) > ops.Policy.MaxPipelineDepth {
		return fmt.Sprintf("execute: pipeline depth exceeds limit (%d)", ops.Policy.MaxPipelineDepth), contract.ExitCodeGeneral
	}
	current := ""
	hasInput := false
	for _, command := range pipeline.commands {
		input := current
		inputAvailable := hasInput

		var errMsg string
		var code int
		input, inputAvailable, errMsg, code = applyInputRedirections(ctx, command.args[0], command.redirs, input, inputAvailable, ops)
		if code != 0 {
			return errMsg, code
		}

		out, execCode := e.runCommand(ctx, command.args, input, inputAvailable, ops)
		if execCode != 0 {
			return out, execCode
		}

		out, errMsg, code = applyOutputRedirections(ctx, command.args[0], command.redirs, out, ops)
		if code != 0 {
			return errMsg, code
		}
		current = out
		hasInput = true

		if ops.Policy.MaxOutputBytes > 0 && len(current) > ops.Policy.MaxOutputBytes {
			return fmt.Sprintf("execute: pipeline intermediate result exceeds limit (%d bytes)", ops.Policy.MaxOutputBytes), contract.ExitCodeGeneral
		}
	}
	return current, 0
}

func applyInputRedirections(ctx context.Context, commandName string, redirs []commandRedirect, input string, hasInput bool, ops contract.Ops) (string, bool, string, int) {
	currentInput := input
	currentHasInput := hasInput
	for _, redir := range redirs {
		switch redir.kind {
		case redirectInputHereDoc:
			currentInput = redir.hereDocBody
			currentHasInput = true
		case redirectInputFile:
			if isNullDevice(redir.target) {
				currentInput = ""
				currentHasInput = true
				continue
			}
			pathValue, err := ops.RequireAbsolutePath(redir.target)
			if err != nil {
				return "", false, fmt.Sprintf("%s: %v", commandName, err), contract.ExitCodeUsage
			}
			raw, err := ops.ReadRawContent(ctx, pathValue)
			if err != nil {
				return "", false, fmt.Sprintf("%s: %v", commandName, err), contract.ExitCodeGeneral
			}
			currentInput = raw
			currentHasInput = true
		}
	}
	return currentInput, currentHasInput, "", 0
}

func applyOutputRedirections(ctx context.Context, commandName string, redirs []commandRedirect, output string, ops contract.Ops) (string, string, int) {
	lastOutputRedir := -1
	for idx, redir := range redirs {
		if redir.kind == redirectOutputWrite || redir.kind == redirectOutputAppend {
			lastOutputRedir = idx
		}
	}
	if lastOutputRedir == -1 {
		return output, "", 0
	}

	if !ops.Policy.AllowWrite() {
		return "", "redirection: write is not allowed by policy", contract.ExitCodeUnsupported
	}
	if ops.Policy.WriteMode == contract.WriteModeWriteLimited && ops.Policy.MaxWriteBytes > 0 && len(output) > ops.Policy.MaxWriteBytes {
		return "", fmt.Sprintf("redirection: write exceeds limit (%d bytes)", ops.Policy.MaxWriteBytes), contract.ExitCodeGeneral
	}

	redir := redirs[lastOutputRedir]
	if isNullDevice(redir.target) {
		return "", "", 0
	}
	pathValue, err := ops.RequireAbsolutePath(redir.target)
	if err != nil {
		return "", fmt.Sprintf("%s: %v", commandName, err), contract.ExitCodeUsage
	}

	switch redir.kind {
	case redirectOutputWrite:
		if err := ops.WriteFile(ctx, pathValue, output); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "", "redirection: write is not supported", contract.ExitCodeUnsupported
			}
			return "", fmt.Sprintf("%s: %v", commandName, err), contract.ExitCodeGeneral
		}
	case redirectOutputAppend:
		if ops.AppendFile == nil {
			return "", "redirection: append is not supported", contract.ExitCodeUnsupported
		}
		if err := ops.AppendFile(ctx, pathValue, output); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "", "redirection: append is not supported", contract.ExitCodeUnsupported
			}
			return "", fmt.Sprintf("%s: %v", commandName, err), contract.ExitCodeGeneral
		}
	}
	return "", "", 0
}

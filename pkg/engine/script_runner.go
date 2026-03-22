package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

const maxExecuteStatements = 64

type execOutput struct {
	stdout string
	stderr string
	code   int
}

func (e *Engine) runScript(ctx context.Context, cmdline string, ops contract.Ops) execOutput {
	if err := executionContextErr(ctx); err != nil {
		return contextExecOutput(err)
	}
	statements, err := parseScript(cmdline)
	if err != nil {
		return execOutput{stdout: fmt.Sprintf("execute: %v", err), code: contract.ExitCodeUsage}
	}
	if len(statements) == 0 {
		return execOutput{}
	}
	return e.executeStatements(ctx, statements, ops)
}

func (e *Engine) executeStatements(ctx context.Context, statements []parsedStatement, ops contract.Ops) execOutput {
	if err := executionContextErr(ctx); err != nil {
		return contextExecOutput(err)
	}
	if len(statements) > maxExecuteStatements {
		return execOutput{stdout: fmt.Sprintf("execute: statement count exceeds limit (%d)", maxExecuteStatements), code: contract.ExitCodeGeneral}
	}
	lastResult := execOutput{}
	executed := false

	for _, statement := range statements {
		if err := executionContextErr(ctx); err != nil {
			return contextExecOutput(err)
		}
		statementResult := execOutput{}
		statementExecuted := false

		for idx, node := range statement.pipelines {
			if err := executionContextErr(ctx); err != nil {
				return contextExecOutput(err)
			}
			shouldRun := true
			if idx > 0 {
				switch node.op {
				case "&&":
					shouldRun = statementResult.code == 0
				case "||":
					shouldRun = statementResult.code != 0
				}
			}
			if !shouldRun {
				continue
			}

			statementResult = e.executePipeline(ctx, node.pipeline, ops)
			statementExecuted = true
		}

		if statementExecuted {
			lastResult = statementResult
			executed = true
		}
	}

	if !executed {
		return execOutput{}
	}
	return lastResult
}

func (e *Engine) executePipeline(ctx context.Context, pipeline parsedPipeline, ops contract.Ops) execOutput {
	if err := executionContextErr(ctx); err != nil {
		return contextExecOutput(err)
	}
	if ops.Policy.MaxPipelineDepth > 0 && len(pipeline.commands) > ops.Policy.MaxPipelineDepth {
		return execOutput{stdout: fmt.Sprintf("execute: pipeline depth exceeds limit (%d)", ops.Policy.MaxPipelineDepth), code: contract.ExitCodeGeneral}
	}
	current := ""
	hasInput := false
	pipelineStderr := ""
	for _, command := range pipeline.commands {
		if err := executionContextErr(ctx); err != nil {
			return contextExecOutput(err)
		}
		input := current
		inputAvailable := hasInput

		var errMsg string
		var code int
		input, inputAvailable, errMsg, code = applyInputRedirections(ctx, command.args[0], command.redirs, input, inputAvailable, ops)
		if code != 0 {
			return execOutput{stdout: errMsg, stderr: pipelineStderr, code: code}
		}

		result := e.runCommand(ctx, command.args, input, inputAvailable, ops)
		pipelineStderr = appendOutputText(pipelineStderr, result.stderr)
		if result.code != 0 {
			result.stderr = pipelineStderr
			return result
		}

		var out string
		out, errMsg, code = applyOutputRedirections(ctx, command.args[0], command.redirs, result.stdout, ops)
		if code != 0 {
			return execOutput{stdout: errMsg, stderr: pipelineStderr, code: code}
		}
		current = out
		hasInput = true

		if ops.Policy.MaxOutputBytes > 0 && len(current) > ops.Policy.MaxOutputBytes {
			markTraceOutputTruncated(ctx)
			return execOutput{
				stdout: fmt.Sprintf("execute: pipeline intermediate result exceeds limit (%d bytes)", ops.Policy.MaxOutputBytes),
				stderr: pipelineStderr,
				code:   contract.ExitCodeGeneral,
			}
		}
	}
	return execOutput{stdout: current, stderr: pipelineStderr, code: 0}
}

func applyInputRedirections(ctx context.Context, commandName string, redirs []commandRedirect, input string, hasInput bool, ops contract.Ops) (string, bool, string, int) {
	if err := executionContextErr(ctx); err != nil {
		return "", false, contextExecOutput(err).stdout, contract.ExitCodeGeneral
	}
	currentInput := input
	currentHasInput := hasInput
	for _, redir := range redirs {
		if err := executionContextErr(ctx); err != nil {
			return "", false, contextExecOutput(err).stdout, contract.ExitCodeGeneral
		}
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
	if err := executionContextErr(ctx); err != nil {
		return "", contextExecOutput(err).stdout, contract.ExitCodeGeneral
	}
	lastOutputRedir := -1
	for idx, redir := range redirs {
		if err := executionContextErr(ctx); err != nil {
			return "", contextExecOutput(err).stdout, contract.ExitCodeGeneral
		}
		if redir.kind == redirectOutputWrite || redir.kind == redirectOutputAppend {
			lastOutputRedir = idx
		}
	}
	if lastOutputRedir == -1 {
		return output, "", 0
	}

	markTraceRequestedPath(ctx, redirs[lastOutputRedir].target)
	if !ops.Policy.AllowWrite() {
		markTraceDeniedPath(ctx, redirs[lastOutputRedir].target)
		return "", "redirection: write is not allowed by policy", contract.ExitCodeUnsupported
	}
	if ops.Policy.WriteMode == contract.WriteModeWriteLimited && ops.Policy.MaxWriteBytes > 0 && len(output) > ops.Policy.MaxWriteBytes {
		markTraceDeniedPath(ctx, redirs[lastOutputRedir].target)
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

func appendOutputText(base string, addition string) string {
	switch {
	case strings.TrimSpace(base) == "":
		return addition
	case strings.TrimSpace(addition) == "":
		return base
	case strings.HasSuffix(base, "\n"):
		return base + addition
	default:
		return base + "\n" + addition
	}
}

func executionContextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func contextExecOutput(err error) execOutput {
	if err == nil {
		return execOutput{}
	}
	return execOutput{
		stdout: fmt.Sprintf("execute: %v", err),
		code:   contract.ExitCodeGeneral,
	}
}

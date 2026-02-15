package engine

import (
	"fmt"
	"strings"
)

const (
	maxHereDocBytes   = 4 << 20 // 4MB
	maxStatementCount = 1024
)

func parseScript(input string) ([]parsedStatement, error) {
	tokens, hereDocBodies, err := tokenizeScript(input)
	if err != nil {
		return nil, err
	}
	return parseStatements(tokens, hereDocBodies)
}

func tokenizeScript(input string) ([]scriptToken, []string, error) {
	tokens := make([]scriptToken, 0)
	hereDocBodies := make([]string, 0)
	pendingHereDocDelims := make([]string, 0)
	for i := 0; i < len(input); {
		i = skipInlineSpaces(input, i)
		if i >= len(input) {
			break
		}

		switch input[i] {
		case '\n':
			tokens = append(tokens, scriptToken{kind: scriptTokenSemi, value: ";"})
			i++
			if len(pendingHereDocDelims) > 0 {
				for _, delim := range pendingHereDocDelims {
					body, next, err := readHereDocBody(input, i, delim)
					if err != nil {
						return nil, nil, err
					}
					hereDocBodies = append(hereDocBodies, body)
					i = next
				}
				pendingHereDocDelims = pendingHereDocDelims[:0]
			}
			continue
		case ';':
			tokens = append(tokens, scriptToken{kind: scriptTokenSemi, value: ";"})
			i++
			continue
		case '&':
			if i+1 < len(input) && input[i+1] == '&' {
				tokens = append(tokens, scriptToken{kind: scriptTokenAnd, value: "&&"})
				i += 2
				continue
			}
			return nil, nil, fmt.Errorf("unsupported operator '&'")
		case '|':
			if i+1 < len(input) && input[i+1] == '|' {
				tokens = append(tokens, scriptToken{kind: scriptTokenOr, value: "||"})
				i += 2
				continue
			}
			tokens = append(tokens, scriptToken{kind: scriptTokenPipe, value: "|"})
			i++
			continue
		case '>':
			if i+1 < len(input) && input[i+1] == '>' {
				tokens = append(tokens, scriptToken{kind: scriptTokenRedirAppend, value: ">>"})
				i += 2
				continue
			}
			tokens = append(tokens, scriptToken{kind: scriptTokenRedirOut, value: ">"})
			i++
			continue
		case '<':
			if i+1 < len(input) && input[i+1] == '<' {
				tokens = append(tokens, scriptToken{kind: scriptTokenRedirHereDoc, value: "<<"})
				i += 2
				i = skipInlineSpaces(input, i)
				if i >= len(input) || input[i] == '\n' {
					return nil, nil, fmt.Errorf("heredoc delimiter is required")
				}
				delim, next, err := readShellWord(input, i)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid heredoc delimiter: %v", err)
				}
				tokens = append(tokens, scriptToken{kind: scriptTokenWord, value: delim})
				pendingHereDocDelims = append(pendingHereDocDelims, delim)
				i = next
				continue
			}
			tokens = append(tokens, scriptToken{kind: scriptTokenRedirIn, value: "<"})
			i++
			continue
		default:
			word, next, err := readShellWord(input, i)
			if err != nil {
				return nil, nil, err
			}
			tokens = append(tokens, scriptToken{kind: scriptTokenWord, value: word})
			i = next
		}
	}
	if len(pendingHereDocDelims) > 0 {
		return nil, nil, fmt.Errorf("unterminated heredoc, missing delimiter %q", pendingHereDocDelims[0])
	}
	return tokens, hereDocBodies, nil
}

func parseStatements(tokens []scriptToken, hereDocBodies []string) ([]parsedStatement, error) {
	statements := make([]parsedStatement, 0)
	tokenIdx := 0
	hereDocIdx := 0

	for tokenIdx < len(tokens) {
		for tokenIdx < len(tokens) && tokens[tokenIdx].kind == scriptTokenSemi {
			tokenIdx++
		}
		if tokenIdx >= len(tokens) {
			break
		}

		firstPipeline, nextIdx, nextHereDocIdx, err := parsePipelineExpr(tokens, tokenIdx, hereDocBodies, hereDocIdx)
		if err != nil {
			return nil, err
		}
		tokenIdx = nextIdx
		hereDocIdx = nextHereDocIdx

		statement := parsedStatement{
			pipelines: []conditionalPipeline{{
				op:       "",
				pipeline: firstPipeline,
			}},
		}

		for tokenIdx < len(tokens) && (tokens[tokenIdx].kind == scriptTokenAnd || tokens[tokenIdx].kind == scriptTokenOr) {
			opToken := tokens[tokenIdx]
			tokenIdx++

			pipeline, pipeNextIdx, pipeHereDocIdx, err := parsePipelineExpr(tokens, tokenIdx, hereDocBodies, hereDocIdx)
			if err != nil {
				return nil, err
			}
			tokenIdx = pipeNextIdx
			hereDocIdx = pipeHereDocIdx

			statement.pipelines = append(statement.pipelines, conditionalPipeline{
				op:       opToken.value,
				pipeline: pipeline,
			})
		}

		if tokenIdx < len(tokens) && tokens[tokenIdx].kind != scriptTokenSemi {
			return nil, fmt.Errorf("unexpected token %q", tokens[tokenIdx].value)
		}
		statements = append(statements, statement)
		if len(statements) > maxStatementCount {
			return nil, fmt.Errorf("statement count exceeds limit (%d)", maxStatementCount)
		}
	}

	if hereDocIdx != len(hereDocBodies) {
		return nil, fmt.Errorf("heredoc bodies not fully consumed")
	}
	return statements, nil
}

func parsePipelineExpr(tokens []scriptToken, tokenIdx int, hereDocBodies []string, hereDocIdx int) (parsedPipeline, int, int, error) {
	command, nextIdx, nextHereDocIdx, err := parseSimpleCommand(tokens, tokenIdx, hereDocBodies, hereDocIdx)
	if err != nil {
		return parsedPipeline{}, tokenIdx, hereDocIdx, err
	}
	pipeline := parsedPipeline{commands: []parsedCommand{command}}
	tokenIdx = nextIdx
	hereDocIdx = nextHereDocIdx

	for tokenIdx < len(tokens) && tokens[tokenIdx].kind == scriptTokenPipe {
		tokenIdx++
		command, nextIdx, nextHereDocIdx, err = parseSimpleCommand(tokens, tokenIdx, hereDocBodies, hereDocIdx)
		if err != nil {
			return parsedPipeline{}, tokenIdx, hereDocIdx, err
		}
		pipeline.commands = append(pipeline.commands, command)
		tokenIdx = nextIdx
		hereDocIdx = nextHereDocIdx
	}
	return pipeline, tokenIdx, hereDocIdx, nil
}

func parseSimpleCommand(tokens []scriptToken, tokenIdx int, hereDocBodies []string, hereDocIdx int) (parsedCommand, int, int, error) {
	command := parsedCommand{
		args:   make([]string, 0),
		redirs: make([]commandRedirect, 0),
	}

	for tokenIdx < len(tokens) {
		token := tokens[tokenIdx]
		switch token.kind {
		case scriptTokenWord:
			command.args = append(command.args, token.value)
			tokenIdx++
		case scriptTokenRedirIn, scriptTokenRedirOut, scriptTokenRedirAppend, scriptTokenRedirHereDoc:
			tokenIdx++
			if tokenIdx >= len(tokens) || tokens[tokenIdx].kind != scriptTokenWord {
				return parsedCommand{}, tokenIdx, hereDocIdx, fmt.Errorf("redirection %s requires target", token.value)
			}
			target := tokens[tokenIdx].value
			tokenIdx++
			redir := commandRedirect{target: target}
			switch token.kind {
			case scriptTokenRedirIn:
				redir.kind = redirectInputFile
			case scriptTokenRedirOut:
				redir.kind = redirectOutputWrite
			case scriptTokenRedirAppend:
				redir.kind = redirectOutputAppend
			case scriptTokenRedirHereDoc:
				redir.kind = redirectInputHereDoc
				if hereDocIdx >= len(hereDocBodies) {
					return parsedCommand{}, tokenIdx, hereDocIdx, fmt.Errorf("missing heredoc body for delimiter %q", target)
				}
				redir.hereDocBody = hereDocBodies[hereDocIdx]
				hereDocIdx++
			}
			command.redirs = append(command.redirs, redir)
		case scriptTokenPipe, scriptTokenAnd, scriptTokenOr, scriptTokenSemi:
			if len(command.args) == 0 {
				return parsedCommand{}, tokenIdx, hereDocIdx, fmt.Errorf("missing command before %s", token.value)
			}
			return command, tokenIdx, hereDocIdx, nil
		default:
			return parsedCommand{}, tokenIdx, hereDocIdx, fmt.Errorf("unexpected token %q", token.value)
		}
	}

	if len(command.args) == 0 {
		return parsedCommand{}, tokenIdx, hereDocIdx, fmt.Errorf("missing command")
	}
	return command, tokenIdx, hereDocIdx, nil
}

func skipInlineSpaces(input string, idx int) int {
	for idx < len(input) {
		switch input[idx] {
		case ' ', '\t', '\r':
			idx++
		default:
			return idx
		}
	}
	return idx
}

func readShellWord(input string, start int) (string, int, error) {
	if start >= len(input) {
		return "", start, fmt.Errorf("expected token")
	}
	var quote byte
	var buf strings.Builder
	escaped := false
	i := start

	for i < len(input) {
		ch := input[i]
		if escaped {
			buf.WriteByte(ch)
			escaped = false
			i++
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
				i++
				continue
			}
			if ch == '\\' {
				escaped = true
				i++
				continue
			}
			buf.WriteByte(ch)
			i++
			continue
		}
		switch ch {
		case '\\':
			escaped = true
			i++
		case '\'', '"':
			quote = ch
			i++
		case ' ', '\t', '\r', '\n', '|', '&', ';', '<', '>':
			if buf.Len() == 0 {
				return "", start, fmt.Errorf("expected token")
			}
			return buf.String(), i, nil
		default:
			buf.WriteByte(ch)
			i++
		}
	}

	if quote != 0 {
		return "", start, fmt.Errorf("unclosed quote")
	}
	if escaped {
		buf.WriteByte('\\')
	}
	if buf.Len() == 0 {
		return "", start, fmt.Errorf("expected token")
	}
	return buf.String(), i, nil
}

func readHereDocBody(input string, start int, delim string) (string, int, error) {
	i := start
	var body strings.Builder
	for {
		if i >= len(input) {
			return "", start, fmt.Errorf("unterminated heredoc, missing delimiter %q", delim)
		}
		lineStart := i
		for i < len(input) && input[i] != '\n' {
			i++
		}
		line := input[lineStart:i]
		line = strings.TrimSuffix(line, "\r")
		if line == delim {
			if i < len(input) && input[i] == '\n' {
				i++
			}
			return body.String(), i, nil
		}
		body.WriteString(line)
		if body.Len() > maxHereDocBytes {
			return "", start, fmt.Errorf("heredoc body exceeds size limit (%d bytes)", maxHereDocBytes)
		}
		if i < len(input) && input[i] == '\n' {
			body.WriteByte('\n')
			i++
		}
	}
}

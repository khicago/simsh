package builtin

import (
	"fmt"
	"strconv"
	"strings"
)

type jsonPathStep struct {
	key   string
	index *int
}

func parseJSONPath(raw string) ([]jsonPathStep, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ".")
	steps := make([]jsonPathStep, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("path must not contain empty segments")
		}
		for len(part) > 0 {
			bracket := strings.IndexByte(part, '[')
			if bracket < 0 {
				steps = append(steps, jsonPathStep{key: part})
				part = ""
				continue
			}
			if bracket > 0 {
				steps = append(steps, jsonPathStep{key: part[:bracket]})
			}
			end := strings.IndexByte(part[bracket:], ']')
			if end < 0 {
				return nil, fmt.Errorf("unterminated array index")
			}
			indexText := strings.TrimSpace(part[bracket+1 : bracket+end])
			index, err := strconv.Atoi(indexText)
			if err != nil || index < 0 {
				return nil, fmt.Errorf("invalid array index %q", indexText)
			}
			steps = append(steps, jsonPathStep{index: &index})
			part = part[bracket+end+1:]
		}
	}
	return steps, nil
}

func applyJSONPath(value any, steps []jsonPathStep) (any, error) {
	current := value
	for _, step := range steps {
		if step.key != "" {
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("path expects object for key %q", step.key)
			}
			next, ok := obj[step.key]
			if !ok {
				return nil, fmt.Errorf("path key %q not found", step.key)
			}
			current = next
		}
		if step.index != nil {
			items, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("path expects array for index %d", *step.index)
			}
			if *step.index >= len(items) {
				return nil, fmt.Errorf("array index %d out of range", *step.index)
			}
			current = items[*step.index]
		}
	}
	return current, nil
}

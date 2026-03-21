package builtin

import (
	"embed"
	"sort"
	"strings"
)

//go:embed commands/*/manual.md
var manualFS embed.FS

// LoadEmbeddedManual returns the full markdown content for a command.
func LoadEmbeddedManual(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	data, err := manualFS.ReadFile("commands/" + trimmed + "/manual.md")
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadAllManualNames returns all available manual names.
func LoadAllManualNames() []string {
	entries, err := manualFS.ReadDir("commands")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.TrimSpace(e.Name())
		if name == "" {
			continue
		}
		if _, err := manualFS.ReadFile("commands/" + name + "/manual.md"); err == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// commandExamples provides example strings for each command, usable by spec functions.
var commandExamples = map[string][]string{
	"ls":   {"ls /", "ls -l /task_outputs", "ls -R /knowledge_base"},
	"tree": {"tree /", "tree -L 2 /task_outputs", "tree -a /knowledge_base"},
	"frontmatter": {
		"frontmatter stat docs -r",
		"frontmatter get --key title /knowledge_base/notes.md",
		"frontmatter print --key sop -C 2 docs/must-sop.md",
	},
	"cat":   {"cat /knowledge_base/readme.md", "cat -n /knowledge_base/readme.md", "echo hello | cat"},
	"head":  {"head /task_outputs/report.md", "head -n 5 /task_outputs/report.md"},
	"tail":  {"tail /task_outputs/report.md", "tail -n 5 /task_outputs/report.md"},
	"grep":  {"grep TODO /task_outputs/notes.md", "grep -E \"func\\s+\" -r /knowledge_base", "grep -l error -r /task_outputs"},
	"find":  {"find / -name \"*.md\"", "find /task_outputs -name \"*.json\"", "find /knowledge_base -name \"*.md\" -exec cat {} ;"},
	"echo":  {"echo hello world", "echo \"report data\" | tee /task_outputs/report.txt"},
	"tee":   {"echo content | tee /task_outputs/file.md", "echo more | tee -a /task_outputs/file.md"},
	"sed":   {"sed -i 's/draft/final/' /task_outputs/report.md", "sed -n '10,20p' /knowledge_base/data.txt"},
	"man":   {"man ls", "man -v grep", "man --list"},
	"date":  {"date", "date -u", "date +%F", "date +%s"},
	"env":   {"env", "env PATH"},
	"mkdir": {"mkdir /task_outputs/reports", "mkdir -p /task_outputs/a/b/c"},
	"cp":    {"cp /knowledge_base/template.md /task_outputs/report.md", "cp /task_outputs/input.txt /task_outputs/output.txt"},
	"mv":    {"mv /task_outputs/draft.md /task_outputs/final.md"},
	"rm":    {"rm /task_outputs/old.md", "rm /task_outputs/temp1.txt /task_outputs/temp2.txt"},
	"rmdir": {"rmdir /task_outputs/empty_dir", "rmdir /task_outputs/cache/a /task_outputs/cache/b"},
	"touch": {"touch /task_outputs/notes.md"},
	"cd":    {"cd /task_outputs", "cd ../knowledge_base", "cd"},
	"pwd":   {"pwd"},
	"which": {"which ls", "which report_tool", "which ll"},
	"type":  {"type ls", "type report_tool", "type ll"},
	"wc":    {"wc /task_outputs/report.md", "wc -l /knowledge_base/data.txt"},
	"sort":  {"sort /task_outputs/names.txt", "sort -rn /task_outputs/scores.txt"},
	"uniq":  {"sort /task_outputs/log.txt | uniq -c", "uniq -d /task_outputs/sorted.txt"},
	"diff":  {"diff /knowledge_base/v1.md /knowledge_base/v2.md"},
}

// ExamplesFor returns the example strings for a command.
func ExamplesFor(name string) []string {
	return commandExamples[name]
}

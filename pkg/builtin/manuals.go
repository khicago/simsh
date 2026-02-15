package builtin

import (
	"embed"
	"strings"
)

//go:embed manuals/*.md
var manualFS embed.FS

// LoadEmbeddedManual returns the full markdown content for a command.
func LoadEmbeddedManual(name string) string {
	data, err := manualFS.ReadFile("manuals/" + name + ".md")
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadAllManualNames returns all available manual names.
func LoadAllManualNames() []string {
	entries, err := manualFS.ReadDir("manuals")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".md") {
			names = append(names, strings.TrimSuffix(name, ".md"))
		}
	}
	return names
}

// commandExamples provides example strings for each command, usable by spec functions.
var commandExamples = map[string][]string{
	"ls":    {"ls /", "ls -l /task_outputs", "ls -R /knowledge_base"},
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
	"cp":    {"cp /knowledge_base/template.md /task_outputs/report.md", "cp -r /knowledge_base/templates /task_outputs/templates"},
	"mv":    {"mv /task_outputs/draft.md /task_outputs/final.md"},
	"rm":    {"rm /task_outputs/old.md", "rm -r /task_outputs/old_reports"},
	"touch": {"touch /task_outputs/notes.md"},
	"wc":    {"wc /task_outputs/report.md", "wc -l /knowledge_base/data.txt"},
	"sort":  {"sort /task_outputs/names.txt", "sort -rn /task_outputs/scores.txt"},
	"uniq":  {"sort /task_outputs/log.txt | uniq -c", "uniq -d /task_outputs/sorted.txt"},
	"diff":  {"diff /knowledge_base/v1.md /knowledge_base/v2.md"},
}

// ExamplesFor returns the example strings for a command.
func ExamplesFor(name string) []string {
	return commandExamples[name]
}

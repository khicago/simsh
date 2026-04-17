package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	externalmapping "github.com/khicago/simsh/benchmarks/external_mapping"
	"github.com/khicago/simsh/pkg/contract"
)

type taskExecutionState struct {
	budget PairedTaskBudget
	run    SubstrateRunRecord
}

type stepClassification struct {
	kind                 string
	note                 string
	misunderstandingKind string
	source               string
}

type commandSurfaceSignal struct {
	Missing          bool
	UsedStructured   bool
	OutcomeKind      contract.ExternalOutcomeKind
	CompatibilityHit string
}

func newTaskExecutionState(substrate string, budget PairedTaskBudget) *taskExecutionState {
	return &taskExecutionState{
		budget: budget,
		run: SubstrateRunRecord{
			Substrate:   substrate,
			StepsDetail: make([]StepRecord, 0, budget.MaxSteps),
		},
	}
}

func (s *taskExecutionState) canContinue() bool {
	return s.run.FailureKind == "" && s.run.Steps < s.budget.MaxSteps
}

func (s *taskExecutionState) recordStep(label, command string, result contract.ExecutionResult, class stepClassification) {
	observationBytes := len(result.Stdout) + len(result.Stderr)
	observationTokens := estimateObservationTokens(observationBytes)
	countedAsWasted := false
	step := StepRecord{
		Index:                   s.run.Steps + 1,
		Label:                   label,
		Command:                 command,
		ExitCode:                result.ExitCode,
		ObservationBytes:        observationBytes,
		ApproxObservationTokens: observationTokens,
		Classification:          class.kind,
		ClassificationSource:    strings.TrimSpace(class.source),
		Note:                    strings.TrimSpace(class.note),
		ExternalOutcomes:        summarizeExternalOutcomes(result.Trace.ExternalOutcomes),
	}
	s.run.Steps++
	s.run.DurationMS += result.DurationMS
	s.run.ObservationBytes += observationBytes
	s.run.ApproxObservationTokens += observationTokens
	switch class.kind {
	case stepClassRetry:
		step.Retry = true
		step.Wasted = true
		s.run.Retries++
		s.run.WastedSteps++
		s.run.WastedObservationTokens += observationTokens
		countedAsWasted = true
	case stepClassWasted:
		step.Wasted = true
		s.run.WastedSteps++
		s.run.WastedObservationTokens += observationTokens
		countedAsWasted = true
	case stepClassEnvMisunderstanding:
		step.Retry = true
		step.Wasted = true
		step.EnvironmentMisunderstood = true
		step.MisunderstandingKind = class.misunderstandingKind
		s.run.Retries++
		s.run.WastedSteps++
		s.run.EnvironmentMisunderstandings++
		s.run.WastedObservationTokens += observationTokens
		countedAsWasted = true
		s.run.LastMisunderstandingKind = class.misunderstandingKind
	}
	if s.run.FailureKind == "" && s.run.ApproxObservationTokens > s.budget.MaxObservationTokens {
		if s.run.LastMisunderstandingKind != "" {
			s.run.FailureKind = failureKindBudgetAfterFallback
		} else {
			s.run.FailureKind = failureKindBudgetExhausted
		}
		if !countedAsWasted {
			s.run.WastedObservationTokens += observationTokens
		}
		if strings.TrimSpace(step.Note) == "" {
			step.Note = "step exhausted the observation budget"
		} else {
			step.Note += "; step exhausted the observation budget"
		}
		s.run.Notes = append(s.run.Notes, fmt.Sprintf("observation budget exceeded: got %d tokens, limit %d", s.run.ApproxObservationTokens, s.budget.MaxObservationTokens))
	}
	s.run.StepsDetail = append(s.run.StepsDetail, step)
}

func (s *taskExecutionState) fail(kind, note string) SubstrateRunRecord {
	if s.run.FailureKind == "" {
		s.run.FailureKind = kind
	}
	if trimmed := strings.TrimSpace(note); trimmed != "" {
		s.run.Notes = append(s.run.Notes, trimmed)
	}
	return s.run
}

func (s *taskExecutionState) succeed(note string) SubstrateRunRecord {
	s.run.Success = true
	if trimmed := strings.TrimSpace(note); trimmed != "" {
		s.run.Notes = append(s.run.Notes, trimmed)
	}
	return s.run
}

func estimateObservationTokens(bytes int) int {
	if bytes <= 0 {
		return 0
	}
	return (bytes + 3) / 4
}

func classificationProgress(note string) stepClassification {
	return stepClassification{kind: stepClassProgress, note: note}
}

func classificationWasted(note string) stepClassification {
	return stepClassification{kind: stepClassWasted, note: note}
}

func classificationMisunderstanding(kind, note string) stepClassification {
	return classificationMisunderstandingWithSource(kind, note, "")
}

func classificationMisunderstandingWithSource(kind, note string, source string) stepClassification {
	return stepClassification{
		kind:                 stepClassEnvMisunderstanding,
		note:                 note,
		misunderstandingKind: kind,
		source:               source,
	}
}

func runTaskPair(ctx context.Context, inventory externalmapping.ScenarioInventory, manifest TaskManifest, task PairedTaskManifest, manifestPath string) (PairRunRecord, error) {
	scenario, ok := inventory.LookupScenario(task.ScenarioID)
	if !ok {
		return PairRunRecord{}, fmt.Errorf("scenario %q missing from inventory", task.ScenarioID)
	}
	if !scenarioIsSupported(task.ScenarioID) {
		return PairRunRecord{}, fmt.Errorf("scenario %q is not supported by paired uplift harness", task.ScenarioID)
	}
	record := PairRunRecord{
		ScenarioID:    scenario.ID,
		Category:      scenario.Category,
		TaskShape:     scenario.TaskShape,
		Summary:       scenario.Summary,
		TruthSurfaces: append([]string(nil), scenario.TruthSurfaces...),
		PairSeed:      task.PairSeed,
		RunOrder:      task.RunOrder,
		AgentID:       manifest.AgentID,
		Budget: PairedTaskBudget{
			MaxSteps:             task.MaxSteps,
			MaxObservationTokens: task.MaxObservationTokens,
		},
		ExpectedOutputs: append([]string(nil), task.ExpectedOutputs...),
		WhySelected:     task.WhySelected,
		EvidenceRefs: []string{
			manifestPath + "#" + task.ScenarioID,
			externalmapping.DefaultScenarioInventoryPath + "#" + task.ScenarioID,
		},
	}

	runOne := func(substrateID string) (SubstrateRunRecord, error) {
		hostRoot, err := os.MkdirTemp("", "simsh-k030-*")
		if err != nil {
			return SubstrateRunRecord{}, err
		}
		defer os.RemoveAll(hostRoot)
		if err := seedScenarioHostRoot(hostRoot, task.ScenarioID); err != nil {
			return SubstrateRunRecord{}, err
		}
		substrate, err := newSubstrate(ctx, substrateID, hostRoot)
		if err != nil {
			return SubstrateRunRecord{}, err
		}
		defer substrate.Close(ctx)
		return executeScenarioTask(ctx, substrate, task)
	}

	switch task.RunOrder {
	case pairRunOrderAB:
		simshRun, err := runOne(manifest.SimshSubstrate)
		if err != nil {
			return PairRunRecord{}, err
		}
		baselineRun, err := runOne(manifest.BaselineSubstrate)
		if err != nil {
			return PairRunRecord{}, err
		}
		record.Simsh = simshRun
		record.Baseline = baselineRun
	case pairRunOrderBA:
		baselineRun, err := runOne(manifest.BaselineSubstrate)
		if err != nil {
			return PairRunRecord{}, err
		}
		simshRun, err := runOne(manifest.SimshSubstrate)
		if err != nil {
			return PairRunRecord{}, err
		}
		record.Simsh = simshRun
		record.Baseline = baselineRun
	default:
		return PairRunRecord{}, fmt.Errorf("unsupported run order %q", task.RunOrder)
	}
	return record, nil
}

func executeScenarioTask(ctx context.Context, substrate commandSubstrate, task PairedTaskManifest) (SubstrateRunRecord, error) {
	switch task.ScenarioID {
	case "relative_navigation_session":
		return runRelativeNavigationTask(ctx, substrate, task)
	case "inspect_edit_write_loop":
		return runInspectEditWriteTask(ctx, substrate, task)
	case "trace_consumable_planning":
		return runTracePlanningTask(ctx, substrate, task)
	default:
		return SubstrateRunRecord{}, fmt.Errorf("unsupported paired uplift scenario %q", task.ScenarioID)
	}
}

func runRelativeNavigationTask(ctx context.Context, substrate commandSubstrate, task PairedTaskManifest) (SubstrateRunRecord, error) {
	state := newTaskExecutionState(substrate.ID(), PairedTaskBudget{
		MaxSteps:             task.MaxSteps,
		MaxObservationTokens: task.MaxObservationTokens,
	})
	started := time.Now()

	setupCommand := "mkdir -p /task_outputs/project/docs && echo hello > /task_outputs/project/docs/readme.md && cd /task_outputs/project"
	setupResult, err := substrate.Run(ctx, setupCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	state.recordStep("setup_and_cd", setupCommand, setupResult, classificationProgress("seed task output and enter project directory"))
	if setupResult.ExitCode != 0 {
		return state.fail(failureKindExecutionFailed, "setup step failed"), nil
	}
	if !state.canContinue() {
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.run, nil
	}

	pwdCommand := "pwd"
	pwdResult, err := substrate.Run(ctx, pwdCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	expectedCWD := "/task_outputs/project"
	readCommand := "cat docs/readme.md"
	if strings.TrimSpace(pwdResult.Stdout) == expectedCWD {
		state.recordStep("check_pwd", pwdCommand, pwdResult, classificationProgress("cwd persisted across steps"))
	} else {
		state.recordStep("check_pwd", pwdCommand, pwdResult, classificationMisunderstanding(misunderstandingNoSessionCWD, "cwd did not persist; switching to absolute-path fallback"))
		readCommand = "cat /task_outputs/project/docs/readme.md"
	}
	if !state.canContinue() {
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.run, nil
	}

	readResult, err := substrate.Run(ctx, readCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	if readResult.ExitCode != 0 {
		state.recordStep("read_document", readCommand, readResult, classificationWasted("document read failed"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindExecutionFailed, "final document read failed"), nil
	}
	if strings.TrimSpace(readResult.Stdout) != "hello" {
		state.recordStep("read_document", readCommand, readResult, classificationWasted("document read returned unexpected content"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindUnexpectedOutput, "expected final document content to be hello"), nil
	}
	state.recordStep("read_document", readCommand, readResult, classificationProgress("final document read succeeded"))
	state.run.DurationMS = time.Since(started).Milliseconds()
	if state.run.FailureKind != "" {
		return state.run, nil
	}
	return state.succeed("relative navigation task completed"), nil
}

func runInspectEditWriteTask(ctx context.Context, substrate commandSubstrate, task PairedTaskManifest) (SubstrateRunRecord, error) {
	state := newTaskExecutionState(substrate.ID(), PairedTaskBudget{
		MaxSteps:             task.MaxSteps,
		MaxObservationTokens: task.MaxObservationTokens,
	})
	started := time.Now()

	searchCommand := `rg --fmt jsonl "TODO: tighten contract" /task_outputs/project`
	searchResult, err := substrate.Run(ctx, searchCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}

	var targetPath string
	if signal := classifyCommandSurfaceUnavailable(searchResult, "rg"); signal.Missing {
		state.recordStep("search_target", searchCommand, searchResult, classificationMisunderstandingWithSource(misunderstandingMissingRG, "rg is unavailable; falling back to grep -r", classificationSourceForCommandSurfaceSignal(signal)))
		if !state.canContinue() {
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.run, nil
		}
		fallbackCommand := `grep -r "TODO: tighten contract" /task_outputs/project`
		fallbackResult, err := substrate.Run(ctx, fallbackCommand)
		if err != nil {
			return state.fail(failureKindExecutionFailed, err.Error()), nil
		}
		targetPath = parseGrepFallbackPath(fallbackResult.Stdout)
		if fallbackResult.ExitCode != 0 || targetPath == "" {
			state.recordStep("search_target_fallback", fallbackCommand, fallbackResult, classificationWasted("grep fallback did not locate the target file"))
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.fail(failureKindUnexpectedOutput, "grep fallback did not return a target path"), nil
		}
		state.recordStep("search_target_fallback", fallbackCommand, fallbackResult, classificationProgress("grep fallback located the target file"))
	} else {
		targetPath = parseRGPath(searchResult.Stdout)
		if searchResult.ExitCode != 0 || targetPath == "" {
			state.recordStep("search_target", searchCommand, searchResult, classificationWasted("rg did not locate the target file"))
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.fail(failureKindUnexpectedOutput, "rg did not return a target path"), nil
		}
		state.recordStep("search_target", searchCommand, searchResult, classificationProgress("rg located the target file"))
	}
	if !state.canContinue() {
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.run, nil
	}

	editCommand := fmt.Sprintf("sed -i 's/TODO: tighten contract/DONE: tightened contract/' %s", targetPath)
	editResult, err := substrate.Run(ctx, editCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	if editResult.ExitCode != 0 {
		state.recordStep("edit_target", editCommand, editResult, classificationWasted("edit command failed"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindExecutionFailed, "edit step failed"), nil
	}
	state.recordStep("edit_target", editCommand, editResult, classificationProgress("edit command completed"))
	if !state.canContinue() {
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.run, nil
	}

	readCommand := fmt.Sprintf("cat %s", targetPath)
	readResult, err := substrate.Run(ctx, readCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	if readResult.ExitCode != 0 {
		state.recordStep("read_target", readCommand, readResult, classificationWasted("post-edit read failed"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindExecutionFailed, "post-edit read failed"), nil
	}
	if !strings.Contains(readResult.Stdout, "DONE: tightened contract") {
		state.recordStep("read_target", readCommand, readResult, classificationWasted("post-edit content was not updated"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindUnexpectedOutput, "post-edit output did not contain tightened contract marker"), nil
	}
	state.recordStep("read_target", readCommand, readResult, classificationProgress("post-edit verification succeeded"))
	state.run.DurationMS = time.Since(started).Milliseconds()
	if state.run.FailureKind != "" {
		return state.run, nil
	}
	return state.succeed("inspect/edit/write task completed"), nil
}

func runTracePlanningTask(ctx context.Context, substrate commandSubstrate, task PairedTaskManifest) (SubstrateRunRecord, error) {
	state := newTaskExecutionState(substrate.ID(), PairedTaskBudget{
		MaxSteps:             task.MaxSteps,
		MaxObservationTokens: task.MaxObservationTokens,
	})
	started := time.Now()

	lenCommand := "json len --fmt json /knowledge_base/planning/tasks.json --path tasks"
	lenResult, err := substrate.Run(ctx, lenCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}

	count := 0
	owner := ""
	status := ""
	if signal := classifyCommandSurfaceUnavailable(lenResult, "json"); signal.Missing {
		state.recordStep("read_task_count", lenCommand, lenResult, classificationMisunderstandingWithSource(misunderstandingMissingJSON, "json inspector is unavailable; falling back to full-file read", classificationSourceForCommandSurfaceSignal(signal)))
		if !state.canContinue() {
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.run, nil
		}
		fallbackCommand := "cat /knowledge_base/planning/tasks.json"
		fallbackResult, err := substrate.Run(ctx, fallbackCommand)
		if err != nil {
			return state.fail(failureKindExecutionFailed, err.Error()), nil
		}
		var parseErr error
		count, owner, status, parseErr = parsePlanningFallback(fallbackResult.Stdout)
		if fallbackResult.ExitCode != 0 || parseErr != nil {
			state.recordStep("read_planning_fallback", fallbackCommand, fallbackResult, classificationWasted("full-file fallback did not yield a parseable planning payload"))
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.fail(failureKindUnexpectedOutput, "planning fallback output was not parseable"), nil
		}
		state.recordStep("read_planning_fallback", fallbackCommand, fallbackResult, classificationProgress("full-file fallback yielded planning facts"))
		if !state.canContinue() {
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.run, nil
		}
	} else {
		countValue, parseErr := parseJSONLenCount(lenResult.Stdout)
		if lenResult.ExitCode != 0 || parseErr != nil {
			state.recordStep("read_task_count", lenCommand, lenResult, classificationWasted("json len did not yield a parseable count"))
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.fail(failureKindUnexpectedOutput, "json len output was not parseable"), nil
		}
		count = countValue
		state.recordStep("read_task_count", lenCommand, lenResult, classificationProgress("json len yielded task count"))
		if !state.canContinue() {
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.run, nil
		}

		getCommand := "json get --fmt jsonl /knowledge_base/planning/tasks.json --path tasks[0].owner --path tasks[0].status"
		getResult, err := substrate.Run(ctx, getCommand)
		if err != nil {
			return state.fail(failureKindExecutionFailed, err.Error()), nil
		}
		ownerValue, statusValue, parseErr := parseJSONGetOwnerStatus(getResult.Stdout)
		if getResult.ExitCode != 0 || parseErr != nil {
			state.recordStep("read_first_task_fields", getCommand, getResult, classificationWasted("json get did not yield owner/status fields"))
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.fail(failureKindUnexpectedOutput, "json get output was not parseable"), nil
		}
		owner = ownerValue
		status = statusValue
		state.recordStep("read_first_task_fields", getCommand, getResult, classificationProgress("json get yielded first task fields"))
		if !state.canContinue() {
			state.run.DurationMS = time.Since(started).Milliseconds()
			return state.run, nil
		}
	}

	summaryText := fmt.Sprintf("count=%d owner=%s status=%s", count, owner, status)
	writeCommand := fmt.Sprintf("echo '%s' > /task_outputs/planning-summary.txt", summaryText)
	writeResult, err := substrate.Run(ctx, writeCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	if writeResult.ExitCode != 0 {
		state.recordStep("write_summary", writeCommand, writeResult, classificationWasted("summary write failed"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindExecutionFailed, "summary write failed"), nil
	}
	state.recordStep("write_summary", writeCommand, writeResult, classificationProgress("summary write succeeded"))
	if !state.canContinue() {
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.run, nil
	}

	readCommand := "cat /task_outputs/planning-summary.txt"
	readResult, err := substrate.Run(ctx, readCommand)
	if err != nil {
		return state.fail(failureKindExecutionFailed, err.Error()), nil
	}
	if readResult.ExitCode != 0 {
		state.recordStep("read_summary", readCommand, readResult, classificationWasted("summary read failed"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindExecutionFailed, "summary read failed"), nil
	}
	if strings.TrimSpace(readResult.Stdout) != summaryText {
		state.recordStep("read_summary", readCommand, readResult, classificationWasted("summary content did not match expected facts"))
		state.run.DurationMS = time.Since(started).Milliseconds()
		return state.fail(failureKindUnexpectedOutput, "summary output did not match expected planning facts"), nil
	}
	state.recordStep("read_summary", readCommand, readResult, classificationProgress("summary read succeeded"))
	state.run.DurationMS = time.Since(started).Milliseconds()
	if state.run.FailureKind != "" {
		return state.run, nil
	}
	return state.succeed("trace-planning task completed"), nil
}

func seedScenarioHostRoot(hostRoot, scenarioID string) error {
	switch scenarioID {
	case "relative_navigation_session":
		return nil
	case "inspect_edit_write_loop":
		content := strings.Join([]string{
			"# Checklist",
			"",
			"- TODO: tighten contract",
			"- keep agent-facing output reviewable",
			"",
		}, "\n")
		distractor := "notes about unrelated work\n"
		if err := writeVirtualFile(hostRoot, "/task_outputs/project/checklist.md", content); err != nil {
			return err
		}
		return writeVirtualFile(hostRoot, "/task_outputs/project/notes.txt", distractor)
	case "trace_consumable_planning":
		return writeVirtualFile(hostRoot, "/knowledge_base/planning/tasks.json", buildPlanningFixtureJSON())
	default:
		return fmt.Errorf("unsupported scenario fixture %q", scenarioID)
	}
}

func buildPlanningFixtureJSON() string {
	type taskRow struct {
		ID          string `json:"id"`
		Owner       string `json:"owner"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}
	payload := struct {
		Tasks []taskRow `json:"tasks"`
	}{
		Tasks: make([]taskRow, 0, 48),
	}
	for idx := 0; idx < 48; idx++ {
		owner := "ops"
		status := "ready"
		if idx%3 == 1 {
			owner = "platform"
			status = "review"
		} else if idx%3 == 2 {
			owner = "runtime"
			status = "draft"
		}
		payload.Tasks = append(payload.Tasks, taskRow{
			ID:          fmt.Sprintf("task-%02d", idx),
			Owner:       owner,
			Status:      status,
			Description: strings.Repeat(fmt.Sprintf("task-%02d evidence ", idx), 12),
		})
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}

func writeVirtualFile(hostRoot, virtualPath, content string) error {
	hostPath, err := hostPathForVirtualPath(hostRoot, virtualPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(hostPath, []byte(content), 0o644)
}

func hostPathForVirtualPath(hostRoot, virtualPath string) (string, error) {
	switch {
	case strings.HasPrefix(virtualPath, "/knowledge_base/"):
		return filepath.Join(hostRoot, "knowledge_base", strings.TrimPrefix(virtualPath, "/knowledge_base/")), nil
	case virtualPath == "/knowledge_base":
		return filepath.Join(hostRoot, "knowledge_base"), nil
	case strings.HasPrefix(virtualPath, "/task_outputs/"):
		return filepath.Join(hostRoot, "task_outputs", strings.TrimPrefix(virtualPath, "/task_outputs/")), nil
	case virtualPath == "/task_outputs":
		return filepath.Join(hostRoot, "task_outputs"), nil
	case strings.HasPrefix(virtualPath, "/temp_work/"):
		return filepath.Join(hostRoot, "temp_work", strings.TrimPrefix(virtualPath, "/temp_work/")), nil
	case virtualPath == "/temp_work":
		return filepath.Join(hostRoot, "temp_work"), nil
	default:
		return "", fmt.Errorf("unsupported virtual path %q", virtualPath)
	}
}

func classifyCommandSurfaceUnavailable(result contract.ExecutionResult, command string) commandSurfaceSignal {
	lowerStdout := strings.ToLower(result.Stdout)
	lowerStderr := strings.ToLower(result.Stderr)
	normalizedCommand := strings.ToLower(strings.TrimSpace(command))
	for _, outcome := range result.Trace.ExternalOutcomes {
		if !externalOutcomeMatchesCommand(outcome, normalizedCommand) {
			continue
		}
		switch outcome.OutcomeKind {
		case contract.ExternalOutcomeCommandNotFound, contract.ExternalOutcomeUnsupported:
			return commandSurfaceSignal{
				Missing:        true,
				UsedStructured: true,
				OutcomeKind:    outcome.OutcomeKind,
			}
		default:
			return commandSurfaceSignal{
				UsedStructured: true,
				OutcomeKind:    outcome.OutcomeKind,
			}
		}
	}
	if result.ExitCode == 0 {
		return commandSurfaceSignal{}
	}
	missingTargets := []string{
		strings.ToLower(command + ": not found"),
		strings.ToLower(command + ": not supported"),
	}
	for _, target := range missingTargets {
		if strings.Contains(lowerStdout, target) || strings.Contains(lowerStderr, target) {
			return commandSurfaceSignal{
				Missing:          true,
				CompatibilityHit: target,
			}
		}
	}
	return commandSurfaceSignal{}
}

func classificationSourceForCommandSurfaceSignal(signal commandSurfaceSignal) string {
	if signal.UsedStructured {
		return classificationSourceStructured
	}
	if strings.TrimSpace(signal.CompatibilityHit) != "" {
		return classificationSourceCompatText
	}
	return ""
}

func externalOutcomeMatchesCommand(outcome contract.ExecutionTraceStep, normalizedCommand string) bool {
	if normalizedCommand == "" {
		return false
	}
	candidates := []string{
		outcome.Command,
		strings.TrimSpace(outcome.ResolvedPath),
	}
	if len(outcome.Argv) > 0 {
		candidates = append(candidates, outcome.Argv[0])
	}
	for _, candidate := range candidates {
		if normalizeOutcomeCommand(candidate) == normalizedCommand {
			return true
		}
	}
	return false
}

func normalizeOutcomeCommand(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = strings.TrimPrefix(trimmed, contract.VirtualExternalBinDir+"/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.Contains(trimmed, "/") {
		parts := strings.Split(trimmed, "/")
		trimmed = parts[len(parts)-1]
	}
	return trimmed
}

func summarizeExternalOutcomes(outcomes []contract.ExecutionTraceStep) []ExternalOutcomeSummary {
	if len(outcomes) == 0 {
		return nil
	}
	summaries := make([]ExternalOutcomeSummary, 0, len(outcomes))
	for _, outcome := range outcomes {
		summary := ExternalOutcomeSummary{
			Command:      strings.TrimSpace(outcome.Command),
			ResolvedPath: safeExternalOutcomeResolvedPath(outcome),
			OutcomeKind:  string(outcome.OutcomeKind),
			ExitCode:     cloneInt(outcome.ExitCode),
		}
		if summary.Command == "" && len(outcome.Argv) > 0 {
			summary.Command = strings.TrimSpace(outcome.Argv[0])
		}
		if summary.Command == "" && summary.ResolvedPath == "" && summary.OutcomeKind == "" && summary.ExitCode == nil {
			continue
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func safeExternalOutcomeResolvedPath(outcome contract.ExecutionTraceStep) string {
	resolvedPath := strings.TrimSpace(outcome.ResolvedPath)
	if resolvedPath == "" {
		return ""
	}
	if !strings.ContainsAny(resolvedPath, `/\`) {
		return resolvedPath
	}
	switch {
	case contract.IsCommandPathUnder(resolvedPath, contract.VirtualExternalBinDir):
		if command := normalizeOutcomeCommand(resolvedPath); command != "" {
			return contract.VirtualExternalBinDir + "/" + command
		}
	case contract.IsCommandPathUnder(resolvedPath, contract.VirtualSystemBinDir):
		if command := normalizeOutcomeCommand(resolvedPath); command != "" {
			return contract.VirtualSystemBinDir + "/" + command
		}
	}
	return ""
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func parseGrepFallbackPath(raw string) string {
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 3)
		if len(parts) < 2 {
			continue
		}
		return strings.TrimSpace(parts[0])
	}
	return ""
}

func parseRGPath(raw string) string {
	type rgRecord struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var record rgRecord
		if err := json.Unmarshal([]byte(trimmed), &record); err != nil {
			continue
		}
		if record.Kind == "match" && strings.TrimSpace(record.Path) != "" {
			return record.Path
		}
	}
	return ""
}

func parseJSONLenCount(raw string) (int, error) {
	type payload struct {
		Entries []struct {
			OK     bool   `json:"ok"`
			Length int    `json:"length"`
			Error  string `json:"error"`
		} `json:"entries"`
	}
	var decoded payload
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err != nil {
		return 0, err
	}
	if len(decoded.Entries) != 1 {
		return 0, fmt.Errorf("expected one json len entry, got %d", len(decoded.Entries))
	}
	if !decoded.Entries[0].OK {
		return 0, fmt.Errorf(decoded.Entries[0].Error)
	}
	return decoded.Entries[0].Length, nil
}

func parseJSONGetOwnerStatus(raw string) (string, string, error) {
	type record struct {
		Query string `json:"query"`
		Value any    `json:"value"`
	}
	owner := ""
	status := ""
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var decoded record
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return "", "", err
		}
		switch decoded.Query {
		case "tasks[0].owner":
			value, ok := decoded.Value.(string)
			if !ok {
				return "", "", fmt.Errorf("owner value is not a string")
			}
			owner = value
		case "tasks[0].status":
			value, ok := decoded.Value.(string)
			if !ok {
				return "", "", fmt.Errorf("status value is not a string")
			}
			status = value
		}
	}
	if owner == "" || status == "" {
		return "", "", fmt.Errorf("missing owner/status records")
	}
	return owner, status, nil
}

func parsePlanningFallback(raw string) (int, string, string, error) {
	var decoded struct {
		Tasks []struct {
			Owner  string `json:"owner"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return 0, "", "", err
	}
	if len(decoded.Tasks) == 0 {
		return 0, "", "", fmt.Errorf("no tasks found")
	}
	return len(decoded.Tasks), decoded.Tasks[0].Owner, decoded.Tasks[0].Status, nil
}

func validateTaskManifest(manifest TaskManifest, inventory externalmapping.ScenarioInventory) error {
	if manifest.Version != 1 {
		return fmt.Errorf("unsupported task manifest version %d", manifest.Version)
	}
	if manifest.ComparisonRule != pairedUpliftComparisonRule {
		return fmt.Errorf("unexpected comparison rule %q", manifest.ComparisonRule)
	}
	if manifest.AgentID != pairedProbeAgentID {
		return fmt.Errorf("unexpected agent id %q", manifest.AgentID)
	}
	if manifest.BaselineSubstrate != substrateThinCoreStateless {
		return fmt.Errorf("unexpected baseline substrate %q", manifest.BaselineSubstrate)
	}
	if manifest.SimshSubstrate != substrateSimshFullSessioned {
		return fmt.Errorf("unexpected simsh substrate %q", manifest.SimshSubstrate)
	}
	if len(manifest.Tasks) == 0 {
		return fmt.Errorf("task manifest must contain at least one task")
	}
	seenScenario := map[string]struct{}{}
	seenSeed := map[int64]struct{}{}
	for _, task := range manifest.Tasks {
		if _, ok := inventory.LookupScenario(task.ScenarioID); !ok {
			return fmt.Errorf("scenario %q missing from inventory", task.ScenarioID)
		}
		if !scenarioIsSupported(task.ScenarioID) {
			return fmt.Errorf("scenario %q is not supported by paired uplift harness", task.ScenarioID)
		}
		if _, ok := seenScenario[task.ScenarioID]; ok {
			return fmt.Errorf("duplicate scenario %q in task manifest", task.ScenarioID)
		}
		seenScenario[task.ScenarioID] = struct{}{}
		if _, ok := seenSeed[task.PairSeed]; ok {
			return fmt.Errorf("duplicate pair seed %d in task manifest", task.PairSeed)
		}
		seenSeed[task.PairSeed] = struct{}{}
		if task.RunOrder != pairRunOrderAB && task.RunOrder != pairRunOrderBA {
			return fmt.Errorf("scenario %q has unsupported run order %q", task.ScenarioID, task.RunOrder)
		}
		if task.MaxSteps <= 0 {
			return fmt.Errorf("scenario %q must declare positive max_steps", task.ScenarioID)
		}
		if task.MaxObservationTokens <= 0 {
			return fmt.Errorf("scenario %q must declare positive max_observation_tokens", task.ScenarioID)
		}
	}
	ordered := append([]PairedTaskManifest(nil), manifest.Tasks...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].ScenarioID < ordered[j].ScenarioID
	})
	if len(ordered) != len(supportedScenarioIDs()) {
		return fmt.Errorf("task manifest must stay narrow in first cut: got %d tasks, want %d", len(ordered), len(supportedScenarioIDs()))
	}
	return nil
}

package builtin

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

func mutationStatusesFromRecords(
	paths []string,
	kind contract.MutationKind,
	records []contract.MutationRecord,
	fallback map[string]string,
) ([]mutationPathStatus, error) {
	index := mutationRecordIndex(records)
	out := make([]mutationPathStatus, 0, len(paths))
	for _, pathValue := range paths {
		if len(records) > 0 {
			if _, ok := index[mutationRecordKey(kind, pathValue)]; !ok {
				return nil, fmt.Errorf("mount batch returned incomplete result for %s %s", kind, pathValue)
			}
		}
		status := fallback[pathValue]
		if record, ok := index[mutationRecordKey(kind, pathValue)]; ok {
			recordStatus := strings.TrimSpace(record.Status)
			if recordStatus != "" && !strings.EqualFold(recordStatus, "ok") {
				status = record.Status
			}
		}
		if strings.TrimSpace(status) == "" {
			status = "ok"
		}
		out = append(out, mutationPathStatus{Path: pathValue, Status: status})
	}
	return out, nil
}

func mutationBytesFromRecords(
	kind contract.MutationKind,
	pathValue string,
	records []contract.MutationRecord,
	fallback int,
) int {
	index := mutationRecordIndex(records)
	if record, ok := index[mutationRecordKey(kind, pathValue)]; ok && record.BytesWritten > 0 {
		return record.BytesWritten
	}
	return fallback
}

func ensureSuccessfulTransferRecords(expected []contract.MutationSpec, records []contract.MutationRecord) error {
	if len(records) > 0 {
		index := mutationRecordIndex(records)
		for _, op := range expected {
			if _, ok := index[mutationRecordKey(op.Kind, op.Path)]; !ok {
				return fmt.Errorf("mount batch returned incomplete result for %s %s", op.Kind, op.Path)
			}
		}
	}
	for _, record := range records {
		status := strings.TrimSpace(strings.ToLower(record.Status))
		switch status {
		case "", "ok", "written", "created", "removed", "moved":
			continue
		default:
			return fmt.Errorf("mount batch returned non-success status %q for %s", record.Status, record.Path)
		}
	}
	return nil
}

func mutationRecordIndex(records []contract.MutationRecord) map[string]contract.MutationRecord {
	index := make(map[string]contract.MutationRecord, len(records))
	for _, record := range records {
		index[mutationRecordKey(record.Kind, record.Path)] = record
	}
	return index
}

func mutationRecordKey(kind contract.MutationKind, pathValue string) string {
	return string(kind) + "\x00" + pathValue
}

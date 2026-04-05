package engine

import (
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestTraceMutationBytesForRecordFallsBackToRequestPayload(t *testing.T) {
	ops := []contract.MutationSpec{
		{Kind: contract.MutationWriteFile, Path: "/workspace/copied.txt", Content: "hello\nworld\n"},
		{Kind: contract.MutationRemoveFile, Path: "/workspace/readme.md"},
	}

	got := traceMutationBytesForRecord(ops, contract.MutationRecord{
		Kind:   contract.MutationWriteFile,
		Path:   "/workspace/copied.txt",
		Status: "written",
	})
	if got != len("hello\nworld\n") {
		t.Fatalf("traceMutationBytesForRecord(...) = %d, want %d", got, len("hello\nworld\n"))
	}

	got = traceMutationBytesForRecord(ops, contract.MutationRecord{
		Kind: contract.MutationRemoveFile,
		Path: "/workspace/readme.md",
	})
	if got != 0 {
		t.Fatalf("traceMutationBytesForRecord(remove) = %d, want 0", got)
	}
}

func TestRecordMutationBatchTraceFallsBackForMissingRecords(t *testing.T) {
	collector := newExecutionTraceCollector("", contract.Ops{})
	ops := []contract.MutationSpec{
		{Kind: contract.MutationWriteFile, Path: "/workspace/copied.txt", Content: "hello\nworld\n"},
		{Kind: contract.MutationRemoveFile, Path: "/workspace/readme.md"},
	}
	records := []contract.MutationRecord{
		{Kind: contract.MutationWriteFile, Path: "/workspace/copied.txt", Status: "written"},
	}

	recordMutationBatchTrace(collector, ops, records)
	trace := collector.Snapshot()
	if len(trace.WrittenPaths) != 1 || trace.WrittenPaths[0] != "/workspace/copied.txt" {
		t.Fatalf("written paths = %#v, want copied.txt", trace.WrittenPaths)
	}
	if len(trace.RemovedPaths) != 1 || trace.RemovedPaths[0] != "/workspace/readme.md" {
		t.Fatalf("removed paths = %#v, want readme.md via request fallback", trace.RemovedPaths)
	}
	if trace.BytesWritten != len("hello\nworld\n") {
		t.Fatalf("bytes_written=%d, want %d", trace.BytesWritten, len("hello\nworld\n"))
	}
}

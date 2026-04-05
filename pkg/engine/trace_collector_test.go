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

package engine

import (
	"context"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
)

func mutationSpecPathOp(kind contract.MutationKind) (contract.PathOp, error) {
	switch kind {
	case contract.MutationWriteFile, contract.MutationAppend, contract.MutationEdit:
		return contract.PathOpWrite, nil
	case contract.MutationMakeDir:
		return contract.PathOpMkdir, nil
	case contract.MutationRemoveFile, contract.MutationRemoveDir:
		return contract.PathOpRemove, nil
	default:
		return "", fmt.Errorf("unsupported mutation kind %q", kind)
	}
}

func validateMountedMutationBatch(mount contract.VirtualMount, ops []contract.MutationSpec) error {
	for _, op := range ops {
		pathOp, err := mutationSpecPathOp(op.Kind)
		if err != nil {
			return err
		}
		if err := contract.CheckMountPathOp(mount, pathOp); err != nil {
			return err
		}
	}
	_, err := contract.ApplyMountMutations(context.Background(), mount, contract.MutationBatch{})
	return err
}

func validateFilesystemFallbackMutation(
	op contract.MutationSpec,
	writeFile func(context.Context, string, string) error,
	appendFile func(context.Context, string, string) error,
	editFile func(context.Context, string, string, string, bool) error,
	makeDir func(context.Context, string) error,
	removeFile func(context.Context, string) error,
	removeDir func(context.Context, string) error,
) error {
	switch op.Kind {
	case contract.MutationWriteFile:
		if writeFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationAppend:
		if appendFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationEdit:
		if editFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationMakeDir:
		if makeDir == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationRemoveFile:
		if removeFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationRemoveDir:
		if removeDir == nil {
			return contract.ErrUnsupported
		}
	default:
		return fmt.Errorf("unsupported mutation kind %q", op.Kind)
	}
	return nil
}

func applyFilesystemFallbackMutation(
	ctx context.Context,
	op contract.MutationSpec,
	writeFile func(context.Context, string, string) error,
	appendFile func(context.Context, string, string) error,
	editFile func(context.Context, string, string, string, bool) error,
	makeDir func(context.Context, string) error,
	removeFile func(context.Context, string) error,
	removeDir func(context.Context, string) error,
) error {
	switch op.Kind {
	case contract.MutationWriteFile:
		return writeFile(ctx, op.Path, op.Content)
	case contract.MutationAppend:
		return appendFile(ctx, op.Path, op.Content)
	case contract.MutationEdit:
		return editFile(ctx, op.Path, op.OldString, op.NewString, op.ReplaceAll)
	case contract.MutationMakeDir:
		return makeDir(ctx, op.Path)
	case contract.MutationRemoveFile:
		return removeFile(ctx, op.Path)
	case contract.MutationRemoveDir:
		return removeDir(ctx, op.Path)
	default:
		return fmt.Errorf("unsupported mutation kind %q", op.Kind)
	}
}

func mutationRecordKey(kind contract.MutationKind, pathValue string) string {
	return string(kind) + "\x00" + normalizeAbsolutePath(pathValue)
}

func reorderMutationRecords(requestOps []contract.MutationSpec, records []contract.MutationRecord) ([]contract.MutationRecord, error) {
	if len(records) == 0 {
		return nil, nil
	}
	index := make(map[string][]contract.MutationRecord, len(records))
	for _, record := range records {
		key := mutationRecordKey(record.Kind, record.Path)
		index[key] = append(index[key], record)
	}
	ordered := make([]contract.MutationRecord, 0, len(records))
	for _, op := range requestOps {
		key := mutationRecordKey(op.Kind, op.Path)
		queued := index[key]
		if len(queued) == 0 {
			return nil, fmt.Errorf("missing mutation record for %s", op.Path)
		}
		ordered = append(ordered, queued[0])
		index[key] = queued[1:]
	}
	for _, queued := range index {
		if len(queued) > 0 {
			ordered = append(ordered, queued...)
		}
	}
	return ordered, nil
}

package engine

import (
	"context"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

func withPathAccessPolicy(ops contract.Ops) contract.Ops {
	origDescribePath := ops.DescribePath
	ops.DescribePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
		if origDescribePath == nil {
			return contract.PathMeta{}, contract.ErrUnsupported
		}
		meta, err := origDescribePath(ctx, pathValue)
		if err != nil {
			return contract.PathMeta{}, err
		}
		if strings.TrimSpace(meta.Access) == "" {
			meta.Access = contract.PathAccessReadOnly
			if ops.Policy.AllowWrite() {
				meta.Access = contract.PathAccessReadWrite
			}
		}
		meta.Access = contract.NormalizePathAccess(meta.Access)
		meta.Capabilities = contract.NormalizePathCapabilities(meta.Capabilities)
		if !ops.Policy.AllowWrite() || meta.Access == contract.PathAccessReadOnly {
			meta.Access = contract.PathAccessReadOnly
			meta.Capabilities = contract.StripWriteCapabilities(meta.Capabilities)
		}
		return meta, nil
	}
	return ops
}

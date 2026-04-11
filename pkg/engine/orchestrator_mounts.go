package engine

import (
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

func (e *Engine) withMountRouter(ops contract.Ops) (contract.Ops, error) {
	allMounts := make([]contract.VirtualMount, 0, len(ops.VirtualMounts)+2)
	allMounts = append(allMounts, mount.NewSysBinMount(e.registry))
	for _, m := range ops.VirtualMounts {
		if m != nil {
			allMounts = append(allMounts, m)
		}
	}
	if ops.ListExternalCommands != nil {
		allMounts = append(allMounts, mount.NewExternalBinMount(ops.ListExternalCommands, ops.ReadExternalManual))
	}
	router, err := newMountRouter(allMounts)
	if err != nil {
		return ops, err
	}
	if router.isEmpty() {
		return ops, nil
	}
	return router.wrapOps(ops), nil
}

package builtin

import "github.com/khicago/simsh/pkg/engine"

func RegisterDefaults(reg *engine.Registry) {
	if reg == nil {
		return
	}
	reg.MustRegister(applyCommandDocContract(specLS()))
	reg.MustRegister(applyCommandDocContract(specTree()))
	reg.MustRegister(applyCommandDocContract(specCd()))
	reg.MustRegister(applyCommandDocContract(specPwd()))
	reg.MustRegister(applyCommandDocContract(specEnv()))
	reg.MustRegister(applyCommandDocContract(specFrontmatter()))
	reg.MustRegister(applyCommandDocContract(specCat()))
	reg.MustRegister(applyCommandDocContract(specHead()))
	reg.MustRegister(applyCommandDocContract(specTail()))
	reg.MustRegister(applyCommandDocContract(specGrep()))
	reg.MustRegister(applyCommandDocContract(specFind()))
	reg.MustRegister(applyCommandDocContract(specWhich()))
	reg.MustRegister(applyCommandDocContract(specType()))
	reg.MustRegister(applyCommandDocContract(specEcho()))
	reg.MustRegister(applyCommandDocContract(specTee()))
	reg.MustRegister(applyCommandDocContract(specSed()))
	reg.MustRegister(applyCommandDocContract(specMan()))
	reg.MustRegister(applyCommandDocContract(specDate()))
	reg.MustRegister(applyCommandDocContract(specMkdir()))
	reg.MustRegister(applyCommandDocContract(specCp()))
	reg.MustRegister(applyCommandDocContract(specMv()))
	reg.MustRegister(applyCommandDocContract(specRm()))
	reg.MustRegister(applyCommandDocContract(specRmdir()))
	reg.MustRegister(applyCommandDocContract(specTouch()))
	reg.MustRegister(applyCommandDocContract(specWc()))
	reg.MustRegister(applyCommandDocContract(specSort()))
	reg.MustRegister(applyCommandDocContract(specUniq()))
	reg.MustRegister(applyCommandDocContract(specDiff()))
}

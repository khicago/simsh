package builtin

import "github.com/khicago/simsh/pkg/engine"

func RegisterDefaults(reg *engine.Registry) {
	if reg == nil {
		return
	}
	reg.MustRegister(specLS())
	reg.MustRegister(specTree())
	reg.MustRegister(specCd())
	reg.MustRegister(specPwd())
	reg.MustRegister(specEnv())
	reg.MustRegister(specFrontmatter())
	reg.MustRegister(specCat())
	reg.MustRegister(specHead())
	reg.MustRegister(specTail())
	reg.MustRegister(specGrep())
	reg.MustRegister(specFind())
	reg.MustRegister(specWhich())
	reg.MustRegister(specType())
	reg.MustRegister(specEcho())
	reg.MustRegister(specTee())
	reg.MustRegister(specSed())
	reg.MustRegister(specMan())
	reg.MustRegister(specDate())
	reg.MustRegister(specMkdir())
	reg.MustRegister(specCp())
	reg.MustRegister(specMv())
	reg.MustRegister(specRm())
	reg.MustRegister(specRmdir())
	reg.MustRegister(specTouch())
	reg.MustRegister(specWc())
	reg.MustRegister(specSort())
	reg.MustRegister(specUniq())
	reg.MustRegister(specDiff())
}

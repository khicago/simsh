package reference

import "path"

func virtualPath(parts ...string) string {
	return "/" + path.Join(parts...)
}

var (
	referenceRoot   = virtualPath("knowledge_base", "reference")
	resourcesRoot   = virtualPath("resources")
	skillsRoot      = virtualPath("skills")
	memoryRoot      = virtualPath("memory")
	taskOutputsRoot = virtualPath("task_outputs")
)

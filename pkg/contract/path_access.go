package contract

import "sort"

const (
	PathAccessReadOnly  = "ro"
	PathAccessReadWrite = "rw"
)

const (
	PathCapabilityRead     = "read"
	PathCapabilityList     = "list"
	PathCapabilityDescribe = "describe"
	PathCapabilitySearch   = "search"
	PathCapabilityWrite    = "write"
	PathCapabilityAppend   = "append"
	PathCapabilityEdit     = "edit"
	PathCapabilityMkdir    = "mkdir"
	PathCapabilityRemove   = "remove"
)

func NormalizePathAccess(access string) string {
	switch access {
	case PathAccessReadOnly:
		return PathAccessReadOnly
	case PathAccessReadWrite:
		return PathAccessReadWrite
	default:
		return PathAccessReadOnly
	}
}

func NormalizePathCapabilities(caps []string) []string {
	if len(caps) == 0 {
		return []string{PathCapabilityDescribe}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(caps))
	for _, cap := range caps {
		switch cap {
		case PathCapabilityRead, PathCapabilityList, PathCapabilityDescribe, PathCapabilitySearch,
			PathCapabilityWrite, PathCapabilityAppend, PathCapabilityEdit, PathCapabilityMkdir, PathCapabilityRemove:
			if _, ok := seen[cap]; ok {
				continue
			}
			seen[cap] = struct{}{}
			out = append(out, cap)
		}
	}
	if len(out) == 0 {
		out = append(out, PathCapabilityDescribe)
	}
	sort.Strings(out)
	return out
}

func StripWriteCapabilities(caps []string) []string {
	filtered := make([]string, 0, len(caps))
	for _, cap := range caps {
		switch cap {
		case PathCapabilityWrite, PathCapabilityAppend, PathCapabilityEdit, PathCapabilityMkdir, PathCapabilityRemove:
			continue
		default:
			filtered = append(filtered, cap)
		}
	}
	return NormalizePathCapabilities(filtered)
}

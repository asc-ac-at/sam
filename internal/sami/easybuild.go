package easybuild

import (
	"fmt"
	"regexp"
	"strings"
)

// Easystack reprents a file containing multiple easybuild commands.
// From the tool's perspective, the EbVersion is encoded in the Path.
type Easystack struct {
	Path      string
	EbVersion string
}

// NewEasystack constructs an Easystack object and returns it.
// If it fails to match a version from the path, then it throws an error
func NewEasystack(path string) (*Easystack, error) {
	ver := EbVersionFromPath(path)
	if ver == "" {
		return nil, fmt.Errorf(`EbVersionFromPath(%q) matched nothing`, path)
	}
	ebVer := EbModuleName(ver)
	res := &Easystack{
		Path:      path,
		EbVersion: ebVer,
	}
	return res, nil
}

// ebVersionRe matches the "eb_<semver>" marker in an easystack filename,
// e.g. asc_eb_5.2.1-2023a.yaml. Any prefix before "eb_" is allowed.
var ebVersionRe = regexp.MustCompile(`(?:^|[^0-9A-Za-z])eb_(\d+\.\d+\.\d+)-`)

// EbVersionFromPath extracts an easybuild version from the path string.
// Expects the path to use the convesion of passing "eb_<VER>"
func EbVersionFromPath(path string) string {
	res := ebVersionRe.FindString(path)
	res = strings.TrimLeft(res, "_eb")
	res = strings.TrimRight(res, "-")
	return res
}

// EbModuleName is the format the we will use for the Easystack.EbVersion
// It represents an argument that gets passed to the module command
func EbModuleName(ver string) string {
	return fmt.Sprintf("Easybuild/%s", ver)
}

package utils

import (
	"fmt"
	"strings"
)

// RHMIVersion rappresents an integreatly version composed by a base part (2.0.0, 2.0.1, ...)
// and a build part (ER1, RC2, ..) if it's a prerelase version
type RHMIVersion struct {
	base  string
	build string
}

// NewRHMIVersion parse the integreatly version as a string and returns a Version object
func NewRHMIVersion(version string) (*RHMIVersion, error) {

	if version == "" {
		return nil, fmt.Errorf("the version can not be empty")
	}

	p := strings.Split(version, "-")
	switch len(p) {
	case 1:
		return &RHMIVersion{base: p[0], build: ""}, nil
	case 2:
		if p[1] == "" {
			return nil, fmt.Errorf("the build part of the version %s is empty", version)
		}

		return &RHMIVersion{base: p[0], build: p[1]}, nil
	default:
		return nil, fmt.Errorf("the version %s is invalid", version)
	}
}

func (v *RHMIVersion) String() string {
	p := []string{v.base}
	if v.build != "" {
		p = append(p, v.build)
	}
	return strings.Join(p, "-")
}

// IsPreRrelease returns true if the version end with -ER1, -RC1, ...
func (v *RHMIVersion) IsPreRrelease() bool {
	return v.build != ""
}

func (v *RHMIVersion) ReleaseBranchName() string {
	return fmt.Sprintf("release-v%s", v.base)
}

func (v *RHMIVersion) TagName() string {
	return fmt.Sprintf("v%s", v.String())
}

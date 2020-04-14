package release

import (
	"fmt"
	"strings"
)

type ReleaseVersion struct {
	base  string
	build string
}

// NewReleaseVersion parse the integreatly version as a string and returns a Version object
func NewReleaseVersion(version string) (*ReleaseVersion, error) {
	if version == "" {
		return nil, fmt.Errorf("the version can not be empty")
	}

	p := strings.Split(version, "-")
	switch len(p) {
	case 1:
		return &ReleaseVersion{base: p[0], build: ""}, nil
	case 2:
		if p[1] == "" {
			return nil, fmt.Errorf("the build part of the version %s is empty", version)
		}

		return &ReleaseVersion{base: p[0], build: p[1]}, nil
	default:
		return nil, fmt.Errorf("the version %s is invalid", version)
	}
}

func (v *ReleaseVersion) String() string {
	p := []string{v.base}
	if v.build != "" {
		p = append(p, v.build)
	}
	return strings.Join(p, "-")
}

// IsPreRrelease returns true if the version end with -ER1, -RC1, ...
func (v *ReleaseVersion) IsPreRrelease() bool {
	return v.build != ""
}

func (v *ReleaseVersion) ReleaseBranchName() string {
	return fmt.Sprintf("release-v%s", v.base)
}

func (v *ReleaseVersion) TagName() string {
	return fmt.Sprintf("v%s", v.String())
}

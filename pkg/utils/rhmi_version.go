package utils

import (
	"fmt"
	"strings"
)

const releaseBranchNameTemplate = "prepare-for-release-%s"

// RHMIVersion rappresents an integreatly version composed by a base part (2.0.0, 2.0.1, ...)
// and a build part (ER1, RC2, ..) if it's a prerelase version
type RHMIVersion struct {
	base  string
	build string
	major string
	minor string
	patch string
}

// NewRHMIVersion parse the integreatly version as a string and returns a Version object
func NewRHMIVersion(version string) (*RHMIVersion, error) {

	if version == "" {
		return nil, fmt.Errorf("the version can not be empty")
	}

	p := strings.Split(version, "-")
	switch len(p) {
	case 1:
		parts := strings.Split(p[0], ".")
		return &RHMIVersion{base: p[0], build: "", major: parts[0], minor: parts[1], patch: parts[2]}, nil
	case 2:
		if p[1] == "" {
			return nil, fmt.Errorf("the build part of the version %s is empty", version)
		}
		parts := strings.Split(p[0], ".")
		return &RHMIVersion{base: p[0], build: p[1], major: parts[0], minor: parts[1], patch: parts[2]}, nil
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

// IsPreRelease returns true if the version end with -ER1, -RC1, ...
func (v *RHMIVersion) IsPreRelease() bool {
	return v.build != ""
}

func (v *RHMIVersion) ReleaseBranchName() string {
	return fmt.Sprintf("release-v%s", v.MajorMinor())
}

func (v *RHMIVersion) TagName() string {
	return fmt.Sprintf("v%s", v.String())
}

// RCTagRef returns a git ref that can be used to search for all RC Tags for this version
func (v *RHMIVersion) RCTagRef() string {
	return fmt.Sprintf("v%s-", v.MajorMinorPatch())
}

func (v *RHMIVersion) Base() string {
	return v.base
}

func (v *RHMIVersion) Build() string {
	return v.build
}

func (v *RHMIVersion) InitialPointReleaseTag() string {
	return fmt.Sprintf("v%s.0", v.MajorMinor())
}

func (v *RHMIVersion) MajorMinor() string {
	return fmt.Sprintf("%s.%s", v.major, v.minor)
}

func (v *RHMIVersion) MajorMinorPatch() string {
	return fmt.Sprintf("%s.%s", v.MajorMinor(), v.patch)
}

func (v *RHMIVersion) PolarionReleaseId() string {
	return fmt.Sprintf("v%s_%s_%s", v.major, v.minor, v.patch)
}

func (v *RHMIVersion) PolarionMilestoneId() string {
	return fmt.Sprintf("%s_%s", v.PolarionReleaseId(), v.build)
}

func (v *RHMIVersion) PrepareReleaseBranchName() string {
	return fmt.Sprintf(releaseBranchNameTemplate, v.TagName())
}

func (v *RHMIVersion) IsPatchRelease() bool {
	return v.patch != "0"
}

// Get the image tags that are created by OpenShift CI for the release branch.
// It's "master" for master branch (for all minor release), and "Major.Minor" for all release branches (for patch releases)
func (v *RHMIVersion) ReleaseBranchImageTag() string {
	if v.IsPatchRelease() {
		return v.MajorMinor()
	} else {
		return "master"
	}
}

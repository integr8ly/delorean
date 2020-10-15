package utils

import (
	"fmt"
	"github.com/integr8ly/delorean/pkg/types"
	"strings"
)

const releaseBranchNameTemplate = "prepare-for-release-%s"

// RHMIVersion rappresents an integreatly version composed by a base part (2.0.0, 2.0.1, ...)
// and a build part (ER1, RC2, ..) if it's a prerelase version
type RHMIVersion struct {
	base    string
	build   string
	major   string
	minor   string
	patch   string
	olmType string
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
		return &RHMIVersion{base: p[0], build: "", major: parts[0], minor: parts[1], patch: parts[2], olmType: types.OlmTypeRhmi}, nil
	case 2:
		if p[1] == "" {
			return nil, fmt.Errorf("the build part of the version %s is empty", version)
		}
		parts := strings.Split(p[0], ".")
		return &RHMIVersion{base: p[0], build: p[1], major: parts[0], minor: parts[1], patch: parts[2], olmType: types.OlmTypeRhmi}, nil
	default:
		return nil, fmt.Errorf("the version %s is invalid", version)
	}
}

// NewVersion parse the version as a string based on olmType and returns a Version object
func NewVersion(version string, olmType string) (*RHMIVersion, error) {
	ver, err := NewRHMIVersion(version)
	if err != nil {
		return nil, err
	}
	if ver != nil {
		ver.olmType = olmType
	}
	return ver, nil
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
	switch v.olmType {
	case types.OlmTypeRhmi:
		return fmt.Sprintf("release-v%s", v.MajorMinor())
	case types.OlmTypeRhoam:
		return fmt.Sprintf("rhoam-release-v%s", v.MajorMinor())
	default:
		return fmt.Sprintf("release-v%s", v.MajorMinor())
	}
}

func (v *RHMIVersion) TagName() string {
	switch v.olmType {
	case types.OlmTypeRhmi:
		return fmt.Sprintf("v%s", v.String())
	case types.OlmTypeRhoam:
		return fmt.Sprintf("rhoam-v%s", v.String())
	default:
		return fmt.Sprintf("v%s", v.String())
	}
}

// RCTagRef returns a git ref that can be used to search for all RC Tags for this version
func (v *RHMIVersion) RCTagRef() string {
	switch v.olmType {
	case types.OlmTypeRhmi:
		return fmt.Sprintf("v%s-", v.MajorMinorPatch())
	case types.OlmTypeRhoam:
		return fmt.Sprintf("rhoam-v%s-", v.MajorMinorPatch())
	default:
		return fmt.Sprintf("v%s-", v.MajorMinorPatch())
	}
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

func (v *RHMIVersion) OlmType() string {
	return v.olmType
}

package utils

import (
	"regexp"
	"strings"
)

const (
	osbsRegistry     = "registry-proxy.engineering.redhat.com/rh-osbs"
	DeloreanRegistry = "quay.io/integreatly/delorean"
)

func StripSHAOrTag(in string) string {
	reImage := regexp.MustCompile(`@.*`)
	matched := reImage.FindString(in)
	if matched == "" {
		s := strings.Split(in, ":")
		return s[0] + ":" + s[1] + "_" + s[2]
	}
	out := reImage.ReplaceAllString(in, "_latest")
	return out
}

func BuildDeloreanImage(image string) string {
	s := strings.Split(image, "/")
	if s[0] == "quay.io" && s[1] == "integreatly" {
		i := strings.Split(image, ":")
		image = DeloreanRegistry + i[1]
		return image
	}

	//we need to treat the amq broker image differently
	amq := strings.Split(image, ":")
	imagename := strings.Split(amq[0], "/")
	if imagename[2] == "amq-broker" {
		majorversion := strings.Split(amq[1], ".")
		majmin := strings.Split(majorversion[1], "-")
		image = DeloreanRegistry + ":" + imagename[2] + "-" + majorversion[0] + "-" + imagename[2] + "-" + majorversion[0] + majmin[0] + "-openshift" + ":" + majorversion[0] + "." + majmin[0]
		return image
	}

	image = DeloreanRegistry + ":" + s[1] + "-" + s[2]
	return image
}

func BuildOSBSImage(image string) string {
	s := strings.Split(image, "/")

	//we need to treat the crw operator image differently
	crw := strings.Split(s[2], "@")
	if crw[0] == "crw-2-rhel8-operator" {
		image = osbsRegistry + "/" + s[1] + "-" + "operator" + "@" + crw[1]
		return image
	}

	image = osbsRegistry + "/" + s[1] + "-" + s[2]
	return image
}

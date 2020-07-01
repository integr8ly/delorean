package utils

import (
	"regexp"
	"strings"
)

const (
	osbsRegistry     = "registry-proxy.engineering.redhat.com/rh-osbs"
	DeloreanRegistry = "quay.io/integreatly/delorean"
)

func stripSHAOrTag(in string) string {
	reImage := regexp.MustCompile(`@sha256:.*`)
	shaDigest := reImage.FindString(in)

	if shaDigest == "" {
		s := strings.Split(in, ":")
		if len(s) == 2 {
			//No sha digest or tag
			return s[0] + ":" + s[1] + "_latest"
		}
		//Tag
		return s[0] + ":" + s[1] + "_" + s[2]
	}
	s := strings.Split(in, "@sha256:")
	//Sha digest
	return s[0] + "_" + s[1]
}

func BuildDeloreanImage(image string) string {
	s := strings.Split(image, "/")
	if s[0] == "quay.io" && s[1] == "integreatly" {
		i := strings.Split(image, ":")
		image = DeloreanRegistry + i[1]
		return image
	}
	image = DeloreanRegistry + ":" + s[1] + "-" + s[2]
	return stripSHAOrTag(image)
}

func BuildOSBSImage(image string) string {
	s := strings.Split(image, "/")

	//we need to treat the crw operator image differently
	crw := strings.Split(s[2], "@")
	if crw[0] == "crw-2-rhel8-operator" {
		image = osbsRegistry + "/" + s[1] + "-" + "operator" + "@" + crw[1]
		return image
	}

	if crw[0] == "ose-cli" {
		image = osbsRegistry + "/openshift-ose-cli@" + crw[1]
		return image
	}

	//we need to treat the amq broker image differently
	amq := strings.Split(image, ":")
	imagename := strings.Split(amq[0], "/")
	if imagename[2] == "amq-broker" {
		majorversion := strings.Split(amq[1], ".")
		majmin := strings.Split(majorversion[1], "-")
		image = osbsRegistry + "/" + imagename[2] + "-" + majorversion[0] + "-" + imagename[2] + "-" + majorversion[0] + majmin[0] + "-openshift" + ":" + majorversion[0] + "." + majmin[0]
		return image
	}

	image = osbsRegistry + "/" + s[1] + "-" + s[2]
	return image
}

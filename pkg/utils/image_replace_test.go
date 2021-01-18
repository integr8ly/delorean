package utils

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var envoyImageDtls = *ImageSubs[envoyProxy]
var rateLimitDtls = *ImageSubs[rateLimiting]
var rhssoDlts = *ImageSubs[rhsso]
var opDir = "./testdata/int-op"

func TestGetCurrentEnvoyImageVersion(t *testing.T) {
	// Test1
	envoyImageDtls.FileLocation = func(string) string { return "/pkg/products/threescale/reconciler" }
	currentVersion, _ := envoyImageDtls.GetCurrentVersion(opDir, envoyImageDtls.FileLocation(opDir), envoyImageDtls.LineRegEx)
	if currentVersion.String() != "1.0.0" {
		t.Fatal("currentVersion should be 1.0.0")
	}

	// Test2: Should throw error as image string invalid
	envoyImageDtls.FileLocation = func(string) string { return "/pkg/products/threescale/reconciler-NoVersion" }
	_, err := envoyImageDtls.GetCurrentVersion(opDir, envoyImageDtls.FileLocation(opDir), envoyImageDtls.LineRegEx)
	if err == nil {
		t.Fatal("Error expected as no version in reconciler-NoVersion ")
	}
}

func TestGetNewVersion(t *testing.T) {
	// SemVer pre fixed it "v"
	v, _ := GetNewVersion("image:v1.0")
	if v.String() != "1.0.0" {
		t.Fatal("TestGetNewVersion: Expected 1.0.0, Actual " + v.String())
	}

	v, _ = GetNewVersion("image:v1.0.0-22")
	if v.String() != "1.0.0-22" {
		t.Fatal("TestGetNewVersion: Expected 1.0.0-22, Actual " + v.String())
	}

	v, _ = GetNewVersion("image:v1.0-22.1604567634")
	if v.String() != "1.0.0-22.1604567634" {
		t.Fatal("TestGetNewVersion: Expected 1.0.0-22.1604567634, Actual " + v.String())
	}

	// SemVer not pre fixed it "v"
	v, _ = GetNewVersion("image:1.0")
	if v.String() != "1.0.0" {
		t.Fatal("TestGetNewVersion: Expected 1.0.0, Actual " + v.String())
	}

	v, _ = GetNewVersion("image:1.0.0-22")
	if v.String() != "1.0.0-22" {
		t.Fatal("TestGetNewVersion: Expected 1.0.0-22, Actual " + v.String())
	}

	v, _ = GetNewVersion("image:1.0-22.1604567634")
	if v.String() != "1.0.0-22.1604567634" {
		t.Fatal("TestGetNewVersion: Expected 1.0.0-22.1604567634, Actual " + v.String())
	}

	// SemVer bad format
	_, err := GetNewVersion("image")
	if err == nil {
		t.Fatal("TestGetNewVersion: Error expected")
	}
}

func TestGetCurrentRateLimitingImageVersion(t *testing.T) {
	// Test1
	rateLimitDtls.FileLocation = func(string2 string) string { return "/pkg/products/marin3r/rateLimitService" }
	currentVersion, err := rateLimitDtls.GetCurrentVersion(opDir, rateLimitDtls.FileLocation(opDir), rateLimitDtls.LineRegEx)
	if err != nil {
		t.Fatal("Error getting current version err: ", err)
	}
	if currentVersion.String() != "1.4.0" {
		t.Fatal("currentVersion expected 1.0.0, actual " + currentVersion.String())
	}

	// Test2: Should throw error as image string invalid
	rateLimitDtls.FileLocation = func(string) string { return "/pkg/products/marin3r/rateLimitService-NoImage" }
	currentVersion, err = rateLimitDtls.GetCurrentVersion(opDir, rateLimitDtls.FileLocation(opDir), rateLimitDtls.LineRegEx)
	if err == nil {
		t.Fatal("Error expected as no version in reconciler-NoVersion ")
	}
}

func TestReplaceEnvoyProxyImage(t *testing.T) {
	envoyImageDtls.FileLocation = func(string) string { return "/pkg/products/threescale/reconciler" }
	err := envoyImageDtls.ReplaceImage(opDir, envoyImageDtls.FileLocation(opDir), envoyImageDtls.LineRegEx, "newImage:2.2.2")
	if err != nil {
		t.Fatal("error during replace image. err: ", err)
	}
	// Confirm new Image
	image, _, _ := GetCurrentEnvoyProxyImage(opDir, envoyImageDtls.FileLocation(opDir), envoyImageDtls.LineRegEx)
	if image != "newImage:2.2.2" {
		t.Fatal("New image was not replaced")
	}

	// Replace original and confirm
	err = envoyImageDtls.ReplaceImage(opDir, envoyImageDtls.FileLocation(opDir), envoyImageDtls.LineRegEx, "registry.redhat.io/openshift-service-mesh/proxyv2-rhel8:v1.0")
	if err != nil {
		t.Fatal("error during replace image. err: ", err)
	}
	// Confirm original Image replaced
	image, _, _ = GetCurrentEnvoyProxyImage(opDir, envoyImageDtls.FileLocation(opDir), envoyImageDtls.LineRegEx)
	if image != "registry.redhat.io/openshift-service-mesh/proxyv2-rhel8:v1.0" {
		t.Fatal("Original image was not replaced")
	}
}

func TestReplaceRateLimitingImage(t *testing.T) {
	rateLimitDtls.FileLocation = func(string) string { return "/pkg/products/marin3r/rateLimitService" }
	err := rateLimitDtls.ReplaceImage(opDir, rateLimitDtls.FileLocation(opDir), rateLimitDtls.LineRegEx, "quay.io/integreatly/ratelimit:v2.2.2")
	if err != nil {
		t.Fatal("error during replace image. err: ", err)
	}
	// Confirm new Image
	image, _, _ := GetCurrentRateLimitingImage(opDir, rateLimitDtls.FileLocation(opDir), rateLimitDtls.LineRegEx)
	if image != "quay.io/integreatly/ratelimit:v2.2.2" {
		t.Fatal("New image was not replaced")
	}

	// Replace original and confirm
	err = rateLimitDtls.ReplaceImage(opDir, rateLimitDtls.FileLocation(opDir), rateLimitDtls.LineRegEx, "quay.io/integreatly/ratelimit:v1.4.0")
	if err != nil {
		t.Fatal("error during replace image. err: ", err)
	}
	// Confirm original Image replaced
	image, _, _ = GetCurrentRateLimitingImage(opDir, rateLimitDtls.FileLocation(opDir), rateLimitDtls.LineRegEx)
	if image != "quay.io/integreatly/ratelimit:v1.4.0" {
		t.Fatal("Original image was not replaced")
	}
}

func TestGetRHSSOProductImageFromCSV(t *testing.T) {
	rhssoDlts.FileLocation = func(string) string {
		return opDir + "/manifests/integreatly-rhsso/11.0.3/keycloak-operator.v11.0.3.clusterserviceversion.yaml"
	}
	image, _, _ := GetRHSSOProductImageFromCSV(rhssoDlts.FileLocation(opDir))
	if image != "registry.redhat.io/rh-sso-7/sso74-openshift-rhel8:7.4-8.1604567634" {
		t.Fatal("Original RHSSO image string was not found")
	}
}

func TestGetCurrentRHSSOVersion(t *testing.T) {

	rhssoDlts.FileLocation = func(string) string {
		return opDir + "/manifests/integreatly-rhsso/11.0.3/keycloak-operator.v11.0.3.clusterserviceversion.yaml"
	}

	currentVersion, err := rhssoDlts.GetCurrentVersion(opDir, rhssoDlts.FileLocation(opDir), rhssoDlts.LineRegEx)
	if err != nil {
		t.Fatal(err)
	} else if currentVersion.String() != "7.4.0-8.1604567634" {
		t.Fatal("currentVersion expected 7.4.0-8.1604567634, actual " + currentVersion.String())
	}

	// Test2: Should throw error as image string invalid
	rhssoDlts.FileLocation = func(string) string {
		return opDir + "/manifests/integreatly-rhsso/11.0.3/keycloak-operator.v11.0.3.fake.clusterserviceversion.yaml"
	}
	currentVersion, err = rhssoDlts.GetCurrentVersion(opDir, rhssoDlts.FileLocation(opDir), rhssoDlts.LineRegEx)
	if err == nil {
		t.Fatal("Error expected as no version in reconciler-NoVersion ")
	}
}

func TestGetRHSSOFileLocation(t *testing.T) {
	rhssoDlts = *ImageSubs[rhsso]
	location := rhssoDlts.FileLocation(opDir)
	if location != "./testdata/int-op/manifests/integreatly-rhsso/11.0.3/keycloak-operator.v11.0.3.clusterserviceversion.yaml" {
		t.Fatal("Location was not as expected: ./testdata/int-op/manifests/integreatly-rhsso/11.0.3/keycloak-operator.v11.0.3.clusterserviceversion.yaml, Actual: " + location)
	}
}

func TestReplaceRHSSOImage(t *testing.T) {
	rhssoDlts.FileLocation = func(string) string {
		return opDir + "/manifests/integreatly-rhsso/11.0.3/keycloak-operator.v11.0.3.clusterserviceversion.yaml"
	}
	// Set new image
	err := rhssoDlts.ReplaceImage(opDir, rhssoDlts.FileLocation(opDir), rhssoDlts.LineRegEx, "newImage:10.10.10")
	if err != nil {
		t.Fatal("error during replace image. err: ", err)
	}
	// Confirm new Image
	image, _, _ := GetRHSSOProductImageFromCSV(rhssoDlts.FileLocation(opDir))
	if image != "newImage:10.10.10" {
		t.Fatal("New image was not replaced, Expected: newImage:10.10.10, Actual: ", image)
	}
	// Replace original and confirm
	err = rhssoDlts.ReplaceImage(opDir, rhssoDlts.FileLocation(opDir), rhssoDlts.LineRegEx, "registry.redhat.io/rh-sso-7/sso74-openshift-rhel8:7.4-8.1604567634")
	if err != nil {
		t.Fatal("error during replace image. err: ", err)
	}
	// Confirm original Image replaced
	image, _, _ = GetRHSSOProductImageFromCSV(rhssoDlts.FileLocation(opDir))
	if image != "registry.redhat.io/rh-sso-7/sso74-openshift-rhel8:7.4-8.1604567634" {
		t.Fatal("Original RHSSO image string was not found")
	}
}

func TestCreateMirrorMap(t *testing.T) {

	err := CreateMirrorMap("./testdata/int-op", envoyProxy, "newImage:v1.1-3.333")
	if err != nil {
		t.Fatal("Error during CreateMirrorMap err: ", err)
	}
	// Verify Image Map file exists
	filePath := path.Join("./testdata/int-op", MappingFile)
	if !FileExists(filePath) {
		t.Fatal("Mapping file not created")
	} else {
		// Confirm contents
		read, err := os.Open(filePath)
		if err != nil {
			t.Fatal("Failed to open file")
		}
		bytes, err := ioutil.ReadAll(read)
		if err != nil {
			t.Fatal("Failed to read file")
		}
		if string(bytes) != "newImage:v1.1-3.333 quay.io/integreatly/ews-envoyproxy:1.1.0-3.333" {
			t.Fatal("Mapping file not as expected: newImage:v1.1-3.333 quay.io/integreatly/ews-envoyproxy:1.1.0-3.333, Actual: ", string(bytes))
		}
		err = os.Remove(path.Join("./testdata/int-op", MappingFile))
		if err != nil {
			t.Fatal("Failed to remove file, needs cleanup")
		}
	}
}

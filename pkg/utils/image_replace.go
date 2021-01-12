package utils

import (
	"context"
	"errors"
	"fmt"
	"github.com/blang/semver"
	doctypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/integr8ly/delorean/pkg/types"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	envoyProxy   = "envoyProxy"
	rateLimiting = "rateLimiting"
	rhsso        = "rhsso"
)

type ImageDetails struct {
	FileLocation      func(string) string
	LineRegEx         *regexp.Regexp
	GetCurrentVersion func(string, string, *regexp.Regexp) (*semver.Version, error)
	ReplaceImage      func(string, string, *regexp.Regexp, string) error
	GetOriginImage    func(string, string) (string, error)
	MirrorRepo        string
	OriginRepo        string
}

var validImages = []string{envoyProxy, rateLimiting}

var ImageSubs = map[string]*ImageDetails{
	envoyProxy: {
		FileLocation:      getEnvoyProxyFileLocation,
		LineRegEx:         regexp.MustCompile(`marin3r.3scale.net/envoy-image` + `.*`),
		ReplaceImage:      replaceEnvoyProxyImage,
		GetCurrentVersion: getCurrentEnvoyProxyVersion,
		MirrorRepo:        "quay.io/integreatly/ews-envoyproxy",
	},
	rateLimiting: {
		FileLocation:      getRateLimitingFileLocation,
		LineRegEx:         regexp.MustCompile(`quay.io/integreatly/` + `.*` + `ratelimit` + `.*`),
		GetCurrentVersion: getCurrentRateLimitingVersion,
		ReplaceImage:      replaceRateLimitingImage,
		MirrorRepo:        "quay.io/integreatly/ews-ratelimiting",
		OriginRepo:        "docker.io/envoyproxy/ratelimit",
		GetOriginImage:    GetRateLimitingOriginImage,
	},
	rhsso: {
		FileLocation:      getRHSSOFileLocation,
		LineRegEx:         regexp.MustCompile(""),
		GetCurrentVersion: getCurrentRHSSOVersion,
		ReplaceImage:      replaceRHSSOImage,
		MirrorRepo:        "quay.io/integreatly/ews-rhsso",
	},
}

func GetRateLimitingOriginImage(repo string, imageTag string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println("Error getting docker client: ", err)
		return "", err
	}
	image := repo + ":" + imageTag
	_, err = cli.ImagePull(ctx, image, doctypes.ImagePullOptions{})
	if err != nil {
		fmt.Println("Error pulling image: ", err)
		return "", err
	}
	return image, nil
}

func GetNewVersion(newImage string) (*semver.Version, error) {
	parts := strings.Split(newImage, ":")
	return getVersionFromImageStringSlice(parts, newImage)
}

func getImageStringFromFile(opDir string, fileLocation string, lineRegEx *regexp.Regexp) (string, []byte, error) {
	filePath := opDir + fileLocation
	read, err := os.Open(filePath)
	if err != nil {
		return "", nil, err
	}
	bytes, err := ioutil.ReadAll(read)
	if err != nil {
		return "", nil, err
	}
	return lineRegEx.FindString(string(bytes)), bytes, nil
}

func getVersionFromImageStringSlice(parts []string, imageStr string) (*semver.Version, error) {
	if len(parts) != 2 {
		return nil, errors.New(fmt.Sprintf("Unexpected image string structure %s", imageStr))
	}

	parts = strings.Split(parts[1], "-")
	version, err := semver.ParseTolerant(parts[0])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unexpected image version structure %s, Error: %s", parts[1], err))
	}

	// No build version
	if len(parts) == 1 {
		return &version, nil
	}

	// There is a build version
	newSemVer, err := semver.Parse(version.String() + "-" + parts[1])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unexpected image version structure %s, Error: %s", parts[1], err))
	}

	return &newSemVer, nil
}

func getCurrentEnvoyProxyVersion(opDir string, fileLocation string, lineRegEx *regexp.Regexp) (*semver.Version, error) {
	// example line from integreatly-operator:
	// deploymentConfig.Spec.Template.Annotations["marin3r.3scale.net/envoy-image"] = "registry.redhat.io/openshift-service-mesh/proxyv2-rhel8:2.0"
	// function should return 2.0.0, adding patch

	imageStr, _, err := getImageStringFromFile(opDir, fileLocation, lineRegEx)
	if err != nil {
		return nil, err
	}

	imageStr = strings.ReplaceAll(imageStr, "marin3r.3scale.net/envoy-image\"] = ", "")
	imageStr = strings.ReplaceAll(imageStr, "\"", "")
	parts := strings.Split(imageStr, ":")

	return getVersionFromImageStringSlice(parts, imageStr)
}

func getCurrentRateLimitingVersion(opDir string, fileLocation string, lineRegEx *regexp.Regexp) (*semver.Version, error) {
	// example line from integreatly-operator:
	// Image:   "quay.io/integreatly/ratelimit:v1.4.0",
	// function should return 1.4.0, removing v

	imageStr, _, err := getImageStringFromFile(opDir, fileLocation, lineRegEx)
	if err != nil {
		return nil, err
	}
	imageStr = strings.ReplaceAll(imageStr, "\"", "")
	imageStr = strings.ReplaceAll(imageStr, ",", "")
	parts := strings.Split(imageStr, ":")
	return getVersionFromImageStringSlice(parts, imageStr)
}

func IsValidType(imagetype string) bool {
	for _, t := range validImages {
		if t == imagetype {
			return true
		}
	}
	return false
}

func getEnvoyProxyFileLocation(opDir string) string {
	return "/pkg/products/threescale/reconciler.go"
}

func replaceEnvoyProxyImage(opDir string, fileLocation string, lineRegEx *regexp.Regexp, newImage string) error {
	currentImage, file, err := GetCurrentEnvoyProxyImage(opDir, fileLocation, lineRegEx)
	if err != nil {
		return err
	}

	fmt.Println("Found envoy proxy image to replace: ", currentImage)
	out := strings.Replace(string(file), currentImage, newImage, 1)
	err = ioutil.WriteFile(opDir+fileLocation, []byte(out), 600)
	if err != nil {
		return err
	}

	return nil
}

func GetCurrentEnvoyProxyImage(opDir string, fileLocation string, lineRegEx *regexp.Regexp) (string, []byte, error) {
	line, file, err := getImageStringFromFile(opDir, fileLocation, lineRegEx)
	if err != nil {
		return "", nil, err
	}

	currentImage := regexp.MustCompile(`= "` + `.*` + `"`).FindString(line)
	currentImage = strings.Replace(currentImage, "= \"", "", 1)
	currentImage = strings.Replace(currentImage, "\"", "", 1)
	currentImage = strings.TrimSpace(currentImage)
	if len(currentImage) == 0 {
		return "", nil, errors.New("Failed to find current image to replace")
	}
	return currentImage, file, nil
}

func getRateLimitingFileLocation(opDir string) string {
	return "/pkg/products/marin3r/rateLimitService.go"
}

func replaceRateLimitingImage(opDir string, fileLocation string, lineRegEx *regexp.Regexp, newImage string) error {
	currentImage, file, err := GetCurrentRateLimitingImage(opDir, fileLocation, lineRegEx)
	if err != nil {
		return err
	}
	fmt.Println("Found rate limiting image to replace: ", currentImage)
	out := strings.Replace(string(file), currentImage, newImage, 1)
	err = ioutil.WriteFile(opDir+fileLocation, []byte(out), 600)
	if err != nil {
		return err
	}

	return nil
}

func GetCurrentRateLimitingImage(opDir string, fileLocation string, lineRegEx *regexp.Regexp) (string, []byte, error) {
	line, file, err := getImageStringFromFile(opDir, fileLocation, lineRegEx)
	if err != nil {
		return "", nil, err
	}

	currentImage := strings.Replace(line, "\",", "", 1)
	currentImage = strings.TrimSpace(currentImage)
	if len(currentImage) == 0 {
		return "", nil, errors.New("Failed to find current image to replace")
	}
	return currentImage, file, nil
}

func CreateMirrorMap(directory string, imageType string, originImage string) error {
	newVersion, err := GetNewVersion(originImage)
	if err != nil {
		return err
	}
	dest := ImageSubs[imageType].MirrorRepo + ":" + newVersion.String()
	return WriteToFile(path.Join(directory, MappingFile), []string{fmt.Sprintf("%s %s", originImage, dest)})
}

func getRHSSOFileLocation(opDir string) string {
	// Set the root file
	// get current version from rhsso.package.yaml
	// return the string for the version eg: manifests/integreatly-rhsso/[version]/keycloak-operator.v[version].clusterserviceversion.yaml

	root := opDir + "/manifests/integreatly-rhsso/"
	version, err := getRHSSOVersionFromPackage(root)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%s/keycloak-operator.v%s.clusterserviceversion.yaml", root, version, version)
}

func getCurrentRHSSOVersion(opDir string, fileLocation string, lineRegEx *regexp.Regexp) (*semver.Version, error) {
	// example line from integreatly-operator:
	// value: registry.redhat.io/rh-sso-7/sso74-openshift-rhel8:7.4-8.1604567634
	// function should return 7.4-8.1604567634

	imageStr, _, err := GetRHSSOProductImageFromCSV(fileLocation)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	imageStr = strings.ReplaceAll(imageStr, "\"", "")
	imageStr = strings.ReplaceAll(imageStr, ",", "")
	parts := strings.Split(imageStr, ":")

	return getVersionFromImageStringSlice(parts, imageStr)
}

func replaceRHSSOImage(opDir string, fileLocation string, lineRegEx *regexp.Regexp, newImage string) error {
	currentImage, file, err := GetRHSSOProductImageFromCSV(fileLocation)
	if err != nil {
		return err
	}

	fmt.Println("Found RHSSO image to replace: ", currentImage)
	out := strings.Replace(string(file), currentImage, newImage, 1)
	err = ioutil.WriteFile(fileLocation, []byte(out), 600)
	if err != nil {
		return err
	}

	return nil
}

func GetRHSSOProductImageFromCSV(location string) (string, []byte, error) {
	var manifest types.RhssoManifest

	raw, err := ioutil.ReadFile(location)
	if err != nil {
		fmt.Printf("Unable to locate manifest file: %s\n", location)
		os.Exit(1)
	}

	err = yaml.Unmarshal(raw, &manifest)
	if err != nil {
		fmt.Printf("Unable to parse configuration file: %s\n", err.Error())
		return "", nil, err
	}

	for index, deployment := range manifest.Spec.Install.Spec.Deployments {
		if deployment.Name == "keycloak-operator" {
			for _, envs := range manifest.Spec.Install.Spec.Deployments[index].Spec.Template.Spec.Containers[0].Env {
				if envs.Name == "RELATED_IMAGE_RHSSO_OPENJDK" {
					image := envs.Value
					fileBytes, err := FileAsBytes(location)
					if err != nil {
						return "", nil, err
					}
					return image, fileBytes, nil
				}
			}
		}
	}

	return "", nil, errors.New("No immage was found")
}

func getRHSSOVersionFromPackage(path string) (string, error) {
	filePath := path + "rhsso.package.yaml"
	var packageObj types.RhssoPackage

	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Unable to locate rhsso.package.yaml file.")
		os.Exit(1)
	}

	err = yaml.Unmarshal(raw, &packageObj)
	if err != nil {
		fmt.Printf("Unable to parse configuration file: %s\n", err.Error())
		return "", err
	}

	for _, channel := range packageObj.Channels {
		if channel.Name == "rhmi" {
			currentCSV := strings.Split(channel.CurrentCSV, ".v")
			if len(currentCSV) == 2 {
				return currentCSV[1], nil
			}
		}
	}

	return "", errors.New("Failed to find version currentCSV with vaild string")
}

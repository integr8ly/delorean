package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/blang/semver"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
)

var ReProd = regexp.MustCompile(`registry.redhat.io/.*`)
var ReStage = regexp.MustCompile(`registry.stage.redhat.io/*`)
var ReProxy = regexp.MustCompile(`registry-proxy.engineering.redhat.com/rh-osbs/.*`)
var ReDelorean = regexp.MustCompile(`quay.io/integreatly/delorean.*`)
var imageWhitelist = [1]string{"registry.redhat.io/openshift4/ose-oauth-proxy:4.2"}

const (
	imagePullSecret = "integreatly-delorean-pull-secret"
)

type CSVName struct {
	Name     string
	Version  semver.Version
	Replaces string
}

type relatedImage struct {
	Name  string
	Image string
}

type relatedImageValue struct {
	Name  string
	Value string
}

type CSVNames []CSVName

func (c CSVNames) Len() int           { return len(c) }
func (c CSVNames) Less(i, j int) bool { return c[i].Version.LT(c[j].Version) }
func (c CSVNames) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c CSVNames) Contains(name string) bool {
	for i := 0; i < c.Len(); i++ {
		if c[i].Name == name {
			return true
		}
	}
	return false
}

type CSV struct {
	obj *unstructured.Unstructured
}

func NewCSV(csvFile string) (*CSV, error) {
	source := &unstructured.Unstructured{}
	if err := PopulateObjectFromYAML(csvFile, source); err != nil {
		return nil, err
	}
	return &CSV{obj: source}, nil
}

func (csv *CSV) GetName() string {
	return csv.obj.GetName()
}

func (csv *CSV) GetVersion() (semver.Version, error) {
	version, ok, err := unstructured.NestedString(csv.obj.Object, "spec", "version")
	if !ok || err != nil {
		version = ""
	}
	return semver.Parse(version)
}

func (csv *CSV) GetReplaces() (string, error) {
	r, ok, err := unstructured.NestedString(csv.obj.Object, "spec", "replaces")
	if !ok || err != nil {
		return "", err
	}
	return r, err
}

func (csv *CSV) GetAnnotations() map[string]string {
	return csv.obj.GetAnnotations()
}

func (csv *CSV) SetAnnotations(annotations map[string]string) {
	csv.obj.SetAnnotations(annotations)
}

func (csv *CSV) GetDeploymentSpecs() ([]olmapiv1alpha1.StrategyDeploymentSpec, error) {
	deployments, _, err := unstructured.NestedSlice(csv.obj.Object, "spec", "install", "spec", "deployments")
	if err != nil {
		return nil, err
	}
	ds := make([]olmapiv1alpha1.StrategyDeploymentSpec, len(deployments))
	for k, v := range deployments {
		d := &olmapiv1alpha1.StrategyDeploymentSpec{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(v.(map[string]interface{}), d)
		if err != nil {
			return nil, err
		}
		ds[k] = *d
	}
	return ds, nil
}

func (csv *CSV) SetDeploymentSpecs(deploymentSpecs []olmapiv1alpha1.StrategyDeploymentSpec) error {
	ds := make([]interface{}, len(deploymentSpecs))
	for k, v := range deploymentSpecs {
		s, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v)
		if err != nil {
			return err
		}
		ds[k] = s
	}
	return unstructured.SetNestedSlice(csv.obj.Object, ds, "spec", "install", "spec", "deployments")
}

func (csv *CSV) GetOperatorDeploymentSpec() (*olmapiv1alpha1.StrategyDeploymentSpec, error) {
	deployments, _, err := unstructured.NestedSlice(csv.obj.Object, "spec", "install", "spec", "deployments")
	if err != nil {
		return nil, err
	}
	d, _ := deployments[0].(map[string]interface{})
	o := &olmapiv1alpha1.StrategyDeploymentSpec{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(d, o)
	if err != nil {
		return nil, errors.New("unable to convert object to StrategyDeploymentSpec")
	}
	return o, nil
}

func (csv *CSV) SetOperatorDeploymentSpec(deploymentSpec *olmapiv1alpha1.StrategyDeploymentSpec) error {
	deployments, _, err := unstructured.NestedSlice(csv.obj.Object, "spec", "install", "spec", "deployments")
	if err != nil {
		return err
	}
	deployments[0], err = runtime.DefaultUnstructuredConverter.ToUnstructured(&deploymentSpec)
	if err != nil {
		return err
	}
	return unstructured.SetNestedSlice(csv.obj.Object, deployments, "spec", "install", "spec", "deployments")
}

func (csv *CSV) SetReplaces(replaces string) error {
	return unstructured.SetNestedField(csv.obj.Object, replaces, "spec", "replaces")
}

func (csv *CSV) GetRelatedImages() ([]relatedImage, error) {
	relatedImages, _, err := unstructured.NestedSlice(csv.obj.Object, "spec", "relatedImages")
	if err != nil {
		return nil, err
	}
	ri := make([]relatedImage, len(relatedImages))
	for k, v := range relatedImages {
		i := &relatedImage{}
		vi := &relatedImageValue{}

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(v.(map[string]interface{}), i)
		if err != nil {
			return nil, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(v.(map[string]interface{}), vi)
		if err != nil {
			return nil, err
		}

		if vi.Value != "" {
			i = &relatedImage{
				Name:  vi.Name,
				Image: vi.Value,
			}

		}

		ri[k] = *i
	}

	return ri, nil
}

func (csv *CSV) SetRelatedImages(relatedImages []relatedImage) error {
	ri := make([]interface{}, len(relatedImages))
	for k, v := range relatedImages {
		s, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v)
		if err != nil {
			return err
		}
		ri[k] = s
	}
	return unstructured.SetNestedSlice(csv.obj.Object, ri, "spec", "relatedImages")
}

func (csv *CSV) UpdateEnvVarList(envKeyValMap map[string]string) error {
	deploymentSpecs, err := csv.GetDeploymentSpecs()
	if err != nil {
		return err
	}
	for _, d := range deploymentSpecs {
		for _, c := range d.Spec.Template.Spec.Containers {
			for _, e := range c.Env {
				for k, v := range envKeyValMap {
					if e.Name == k {
						e.ValueFrom.FieldRef.FieldPath = v
					}
				}
			}
		}
	}
	return csv.SetDeploymentSpecs(deploymentSpecs)
}

func (csv *CSV) WriteYAML(yamlFile string) error {
	return WriteK8sObjectToYAML(csv.obj, yamlFile)
}

func (csv *CSV) WriteJSON(jsonFile string) error {
	return WriteObjectToJSON(csv.obj, jsonFile)
}

func GetCSVFileName(csv *CSV) string {
	return fmt.Sprintf("%s.clusterserviceversion.yaml", csv.GetName())
}

// ReadCSVFromBundleDirectory tries to parse every YAML file in the directory and see if they are CSV.
// According to the strict one CSV rule for every bundle, we return the first file that is considered a CSV type.
func ReadCSVFromBundleDirectory(bundleDir string) (*CSV, string, error) {
	dirContent, err := ioutil.ReadDir(bundleDir)
	if err != nil {
		return nil, "", fmt.Errorf("error reading bundle directory %s, %v", bundleDir, err)
	}

	files := []string{}
	for _, f := range dirContent {
		if !f.IsDir() {
			files = append(files, f.Name())
		}
	}

	for _, file := range files {
		if strings.Contains(file, ".clusterserviceversion.yaml") || strings.Contains(file, ".csv.yaml") {
			bundleFilepath := path.Join(bundleDir, file)
			csv, err := NewCSV(bundleFilepath)
			if err != nil {
				return nil, "", err
			}
			return csv, bundleFilepath, nil
		}
	}
	return nil, "", fmt.Errorf("no ClusterServiceVersion object found in %s", bundleDir)

}

func VerifyManifestDirs(dirs ...string) error {
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return err
		}

		_, _, err := GetCurrentCSV(dir)
		if err != nil {
			return err
		}

	}
	return nil
}

func GetPackageManifest(packageDir string) (*registry.PackageManifest, string, error) {
	matches, err := filepath.Glob(packageDir + "/*.package.yaml")
	if err != nil {
		return nil, "", err
	}

	if len(matches) == 0 {
		return nil, "", fmt.Errorf("No package.yaml file found in %s", packageDir)
	}

	var pkgManifestFile = matches[0]

	pkgManifest := &registry.PackageManifest{}
	if err = PopulateObjectFromYAML(pkgManifestFile, &pkgManifest); err != nil {
		return nil, "", err
	}

	return pkgManifest, pkgManifestFile, nil
}

func GetSortedCSVNames(packageDir string) (CSVNames, error) {
	bundleDirs, err := ioutil.ReadDir(packageDir)
	var sortedCSVNames CSVNames
	if err != nil {
		return nil, err
	}
	for _, bundlePath := range bundleDirs {
		if bundlePath.IsDir() {
			csv, _, err := ReadCSVFromBundleDirectory(filepath.Join(packageDir, bundlePath.Name()))
			if err != nil {
				return nil, err
			}
			v, err := csv.GetVersion()
			if err != nil {
				return nil, err
			}
			r, err := csv.GetReplaces()
			if err != nil {
				return nil, err
			}
			sortedCSVNames = append(sortedCSVNames, CSVName{Name: csv.GetName(), Version: v, Replaces: r})
		}
	}
	sort.Sort(sortedCSVNames)
	return sortedCSVNames, nil
}

func GetCurrentCSV(packageDir string) (*CSV, string, error) {

	csv, csvFile, err := ReadCSVFromBundleDirectory(packageDir)
	if err == nil {
		return csv, csvFile, nil
	}

	pkgManifest, _, err := GetPackageManifest(packageDir)
	if err != nil {
		return nil, "", err
	}

	var currentCSVName string
	for _, channel := range pkgManifest.Channels {
		if channel.IsDefaultChannel(*pkgManifest) {
			currentCSVName = channel.CurrentCSVName
			break
		}
	}

	bundleDirs, err := ioutil.ReadDir(packageDir)
	if err != nil {
		return nil, "", fmt.Errorf("error reading from %s directory, %v", packageDir, err)
	}
	for _, bundlePath := range bundleDirs {
		if bundlePath.IsDir() {
			bundleDir := filepath.Join(packageDir, bundlePath.Name())
			csv, csvFile, err := ReadCSVFromBundleDirectory(bundleDir)
			if err != nil {
				return nil, "", err
			}
			if csv.GetName() == currentCSVName {
				return csv, csvFile, nil
			}
		}
	}

	return nil, "", fmt.Errorf("failed to find current csv in %s", packageDir)
}

func UpdatePackageManifest(packageDir, currentCSVName string) (*registry.PackageManifest, error) {

	pkgManifest, pkgManifestFile, err := GetPackageManifest(packageDir)
	if err != nil {
		return nil, err
	}

	pkgManifest.Channels[0].CurrentCSVName = fmt.Sprintf(currentCSVName)
	pkgManifest.DefaultChannelName = pkgManifest.Channels[0].Name

	err = WriteObjectToYAML(pkgManifest, pkgManifestFile)
	if err != nil {
		return nil, err
	}

	return pkgManifest, nil
}

func ProcessCurrentCSV(packageDir string, processFunc process) error {
	csv, csvfile, err := GetCurrentCSV(packageDir)
	if err != nil {
		return err
	}

	if processFunc != nil {
		err = processFunc(csv)
		if err != nil {
			return err
		}
	}

	err = csv.WriteYAML(csvfile)
	if err != nil {
		return err
	}
	return nil
}

type process func(*CSV) error

func GetAndUpdateOperandImages(manifestDir string, extraImages []string, isGa bool) (map[string]string, error) {
	csv, fp, err := GetCurrentCSV(manifestDir)
	if err != nil {
		return nil, err
	}

	if len(extraImages) > 0 {
		if err := includeImages(extraImages, csv); err != nil {
			return nil, err
		}
	}

	var images = map[string]string{}
	deployment, err := csv.GetOperatorDeploymentSpec()
	relatedImages, err := csv.GetRelatedImages()
	if err != nil {
		return nil, err
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			//if it's GA, check for stage images and replace them.
			if isGa {
				stageMatched := ReStage.FindString(env.Value)
				if stageMatched != "" {
					prodImage := processStageToProdImage(env.Value)
					container.Env = AddOrUpdateEnvVar(container.Env, env.Name, prodImage)
				}
				//it it's pre ga check for delorean image and move on
				//check for stage and prod images and replace them with delorean images
			} else {
				deloreanMatched := ReDelorean.FindString(env.Value)
				if deloreanMatched != "" {
					continue
				}
				prodMatched := ReProd.FindString(env.Value)
				stageMatched := ReStage.FindString(env.Value)
				//found a registry.redhat.io image
				if prodMatched != "" || stageMatched != "" {
					deloreanImage := processToDeloreanImage(env.Value)
					mirrorString := BuildOSBSImage(env.Value) + " " + deloreanImage
					addImageMapping(mirrorString, images)
					container.Env = AddOrUpdateEnvVar(container.Env, env.Name, deloreanImage)
				}
			}
		}
	}

	// Check that relatedImages all point to the correct registry if isGa
	if isGa && len(relatedImages) > 0 {
		for k, i := range relatedImages {
			if strings.Contains(i.Image, "registry.stage.redhat.io") {
				relatedImages[k].Image = strings.Replace(i.Image, "registry.stage.redhat.io", "registry.redhat.io", 1)
			}
		}

		err = csv.SetRelatedImages(relatedImages)
		if err != nil {
			return nil, err
		}
	}

	csv.SetOperatorDeploymentSpec(deployment)

	err = csv.WriteYAML(fp)
	if err != nil {
		return images, err
	}
	return images, nil
}

func includeImages(extraImages []string, csv *CSV) error {
	deployment, err := csv.GetOperatorDeploymentSpec()
	if err != nil {
		return err
	}
	for n, container := range deployment.Spec.Template.Spec.Containers {
		for _, i := range extraImages {
			currImage := strings.Split(i, "=")
			container.Env = AddOrUpdateEnvVar(container.Env, currImage[0], currImage[1])
		}
		deployment.Spec.Template.Spec.Containers[n] = container
	}
	csv.SetOperatorDeploymentSpec(deployment)
	return nil
}

func GetAndUpdateOperatorImage(manifestDir string, images map[string]string, isGa bool) (map[string]string, error) {
	csv, fp, err := GetCurrentCSV(manifestDir)
	if err != nil {
		return nil, err
	}

	deployment, err := csv.GetOperatorDeploymentSpec()
	if err != nil {
		return nil, err
	}
	annotations := csv.GetAnnotations()

	//same flow as the above
	//if it's GA, check for stage images and replace them.
	if isGa {
		//check for stage image and replace it if necessary
		stageMatched := ReStage.FindString(deployment.Spec.Template.Spec.Containers[0].Image)
		if stageMatched != "" {
			prodImage := processStageToProdImage(deployment.Spec.Template.Spec.Containers[0].Image)
			deployment.Spec.Template.Spec.Containers[0].Image = prodImage
		}
		stageMatched = ReStage.FindString(annotations["containerImage"])
		if stageMatched != "" {
			prodImage := processStageToProdImage(annotations["containerImage"])
			annotations["containerImage"] = prodImage
		}
	} else {
		//check for stage and prod images and replace with quay image
		stageMatched := ReStage.FindString(deployment.Spec.Template.Spec.Containers[0].Image)
		prodMatched := ReProd.FindString(deployment.Spec.Template.Spec.Containers[0].Image)
		if stageMatched != "" || prodMatched != "" {
			processedDeloreanImage := processToDeloreanImage(deployment.Spec.Template.Spec.Containers[0].Image)
			mirrorString := BuildOSBSImage(deployment.Spec.Template.Spec.Containers[0].Image) + " " + processedDeloreanImage
			addImageMapping(mirrorString, images)
			deployment.Spec.Template.Spec.Containers[0].Image = processedDeloreanImage
		}
		stageMatched = ReStage.FindString(annotations["containerImage"])
		prodMatched = ReProd.FindString(annotations["containerImage"])
		if stageMatched != "" || prodMatched != "" {
			processedDeloreanImage := processToDeloreanImage(annotations["containerImage"])
			annotations["containerImage"] = processedDeloreanImage
		}
	}
	csv.SetAnnotations(annotations)
	csv.SetOperatorDeploymentSpec(deployment)

	err = csv.WriteYAML(fp)
	return images, nil
}

func processToDeloreanImage(image string) string {
	return BuildDeloreanImage(image)
}

func processStageToProdImage(stageImage string) string {
	return strings.Replace(stageImage, ".stage", "", 1)
}

func addImageMapping(mirrorString string, images map[string]string) {
	mapping := strings.Split(mirrorString, " ")
	images[mapping[0]] = mapping[1]
}

func FindDeploymentByName(deployments []olmapiv1alpha1.StrategyDeploymentSpec, name string) (int, *olmapiv1alpha1.StrategyDeploymentSpec) {
	for i, d := range deployments {
		if d.Name == name {
			return i, &d
		}
	}
	return -1, nil
}

func FindContainerByName(containers []corev1.Container, containerName string) (int, *corev1.Container) {
	for i, c := range containers {
		if c.Name == containerName {
			return i, &c
		}
	}
	return -1, nil
}

func FindInstallMode(installModes []olmapiv1alpha1.InstallMode, typeName olmapiv1alpha1.InstallModeType) (int, *olmapiv1alpha1.InstallMode) {
	for i, m := range installModes {
		if m.Type == typeName {
			return i, &m
		}
	}
	return -1, nil
}

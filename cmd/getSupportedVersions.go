package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/blang/semver"
	"github.com/go-git/go-git/v5"
	"github.com/integr8ly/delorean/pkg/types"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type getSupportedVersionsFlags struct {
	olmType                string
	supportedMajorVersions string
	supportedMinorVersions string
	managedTenants         string
}

type getSupportedVersionsCmd struct {
	olmType                string
	supportedMajorVersions int
	supportedMinorVersions int
	managedTenants         string
}

type olmPaths struct {
	bundleFolder  string
	addonFilePath string
}

func init() {
	f := &getSupportedVersionsFlags{}
	cmd := &cobra.Command{
		Use:   "supported-versions",
		Short: "Get a list of supported version of the operator.",
		Long: "Get a list of supported version of the operator. " +
			"The supported versions are compiled from the bundles in the operators folders in managed-tenants. " +
			"Result can be configured to return different number of supported major and minor versions. " +
			"All patch versions are returned for the minor versions.",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newGetSupportedVersions(f)
			if err != nil {
				handleError(err)
			}

			if _, err = c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}

	cmd.Flags().StringVar(&f.olmType, "olmType", types.OlmTypeRhoam, fmt.Sprintf("OlM Type to get the versions of. Supported Values [%s, %s]", types.OlmTypeRhmi, types.OlmTypeRhoam))
	cmd.Flags().StringVarP(&f.supportedMinorVersions, "minor", "m", "3", "Supported number of minor versions")
	cmd.Flags().StringVarP(&f.supportedMajorVersions, "major", "M", "1", "Supported number of major versions")
	cmd.Flags().StringVar(&f.managedTenants, "managedTenants", "https://gitlab.cee.redhat.com/service/managed-tenants.git", "https link for the managed tenants repository to clone")
	pipelineCmd.AddCommand(cmd)

}

func newGetSupportedVersions(f *getSupportedVersionsFlags) (*getSupportedVersionsCmd, error) {
	var majorVersions int
	var minorVersions int

	_, err := fmt.Sscan(f.supportedMajorVersions, &majorVersions)
	_, err = fmt.Sscan(f.supportedMinorVersions, &minorVersions)

	return &getSupportedVersionsCmd{
		olmType:                f.olmType,
		supportedMajorVersions: majorVersions,
		supportedMinorVersions: minorVersions,
		managedTenants:         f.managedTenants,
	}, err
}

func (c *getSupportedVersionsCmd) run(_ context.Context) ([]string, error) {
	paths, err := getOlmTypePaths(c.olmType)
	if err != nil {
		return nil, err
	}

	// Download the manged tenants repo
	repoDir, err := downloadManagedTenants(c.managedTenants)
	if err != nil {
		return nil, err
	}

	// in case of RHOAM bundles would contain no CSV but the reference to an index image with bundles inside
	// Pull and unpack the index
	if c.olmType == types.OlmTypeRhoam {
		err = extractRhoamCSV(&paths, repoDir)
		if err != nil {
			return nil, err
		}
	}

	// get the current production version
	var productionVersion semver.Version
	productionVersion, err = getProductionVersion(repoDir, paths)
	if err != nil {
		return nil, err
	}

	// read the bundle folder names
	bundles, err := getBundleFolders(repoDir, paths.bundleFolder)
	if err != nil {
		return nil, err
	}

	// create a semver object for the folder names
	semverVersions, err := getSemverValues(bundles)
	if err != nil {
		return nil, err

	}

	// trim semver version list to have newest version equal to production version
	semverVersions, err = trimSemverVersions(semverVersions, productionVersion)
	if err != nil {
		return nil, err

	}

	// Get the top major streams versions. Max is the number of version being checked
	majorVersions, err := getMajorVersions(semverVersions, c.supportedMajorVersions)

	// get the top minor streams for the major versions. Limited the max number of versions to be checked
	minorVersion, err := getMinorVersions(semverVersions, majorVersions, c.supportedMinorVersions)
	if err != nil {
		return nil, err
	}

	// For the list of minor minorVersions get a list of all the patch minorVersions
	patchVersions, err := getPatchVersions(semverVersions, minorVersion)
	if err != nil {
		return nil, err

	}

	result := strings.Join(patchVersions, ",")
	fmt.Println(result)
	return patchVersions, nil
}

func extractRhoamCSV(paths *olmPaths, repoDir string) error {
	// get index image
	root := path.Join(repoDir, paths.addonFilePath)

	type indexFiled struct {
		IndexImage string `json:"indexImage"`
	}

	data := indexFiled{}
	err := utils.PopulateObjectFromYAML(root, &data)
	if err != nil {
		return err
	}
	indexSha := data.IndexImage
	// assuming opm is installed
	cmd := exec.Command("opm", "index", "export", fmt.Sprintf("--index=%s", indexSha), fmt.Sprintf("--download-folder=%s", repoDir))
	err = cmd.Run()
	if err != nil {
		return errors.New(fmt.Sprintf("Error when executing \"%s\": %s", cmd.String(), err))
	}
	// update paths with CSV location
	paths.addonFilePath = "managed-api-service/package.yaml"
	return nil
}

func trimSemverVersions(versions []semver.Version, productionVersion semver.Version) ([]semver.Version, error) {
	var result []semver.Version

	for _, version := range versions {
		if version.Major < productionVersion.Major {
			result = append(result, version)
		}

		if version.Major == productionVersion.Major && version.Minor <= productionVersion.Minor {
			result = append(result, version)
		}
	}

	if len(result) == 0 {
		return result, fmt.Errorf("All versions are newer that production")
	}

	return result, nil
}

func getProductionVersion(dir string, paths olmPaths) (semver.Version, error) {
	root := path.Join(dir, paths.addonFilePath)

	type channelType struct {
		Name       string `json:"name"`
		CurrentCSV string `json:"currentCSV"`
	}
	type channelField struct {
		Channels []channelType `json:"channels"`
	}

	data := channelField{}
	err := utils.PopulateObjectFromYAML(root, &data)
	if err != nil {
		return semver.Version{}, err
	}

	version := strings.Split(data.Channels[0].CurrentCSV, ".v")[1]
	semverVersion, err := semver.Make(version)
	if err != nil {
		return semver.Version{}, err
	}
	return semverVersion, nil
}

func downloadManagedTenants(url string) (string, error) {
	dir, err := ioutil.TempDir(os.TempDir(), "managed-tenants")
	if err != nil {
		return "", err
	}

	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL:      url,
		Progress: nil,
	})
	if err != nil {
		return "", err
	}

	return dir, nil
}

func getOlmTypePaths(olmType string) (olmPaths, error) {

	switch olmType {
	case types.OlmTypeRhoam:
		return olmPaths{
			bundleFolder:  "managed-api-service",
			addonFilePath: "addons/rhoams/metadata/production/addon.yaml",
		}, nil
	case types.OlmTypeRhmi:
		return olmPaths{
			bundleFolder:  "addons/integreatly-operator/bundles",
			addonFilePath: "addons/integreatly-operator/metadata/production/addon.yaml",
		}, nil
	default:
		return olmPaths{}, fmt.Errorf("Unsupported OLM type, Please use --help to see supported types.")
	}
}

func getBundleFolders(dir string, bundlePath string) ([]string, error) {
	var bundles []string
	root := path.Join(dir, bundlePath)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if path == root {
				return nil
			}
			_, bundle := filepath.Split(path)
			bundles = append(bundles, bundle)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return bundles, nil
}

func getSemverValues(bundles []string) ([]semver.Version, error) {
	var values []semver.Version
	for _, bundle := range bundles {
		value, err := semver.Make(bundle)

		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

func getMajorVersions(versions []semver.Version, supportedVersions int) ([]int, error) {
	var result []int
	for _, version := range versions {
		if contains(result, int(version.Major)) {
			continue
		}
		result = append(result, int(version.Major))
	}
	sort.Ints(result)
	if len(result) > supportedVersions {
		result = result[len(result)-supportedVersions:]
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("Empty list being returned")
	}

	return result, nil
}

func getMinorVersions(versions []semver.Version, majorVersions []int, supportedVersions int) (map[int][]int, error) {
	var result = make(map[int][]int)
	for _, version := range versions {
		if contains(majorVersions, int(version.Major)) {
			_, exists := result[int(version.Major)]
			if !exists {
				result[int(version.Major)] = []int{}
			}

			if contains(result[int(version.Major)], int(version.Minor)) {
				continue
			}
			result[int(version.Major)] = append(result[int(version.Major)], int(version.Minor))
			sort.Ints(result[int(version.Major)])

			if len(result[int(version.Major)]) > supportedVersions {
				result[int(version.Major)] = result[int(version.Major)][len(result[int(version.Major)])-supportedVersions:]
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("Unexpected error trying to return an  empty map")
	}

	return result, nil
}

func getPatchVersions(versions []semver.Version, supportedVersions map[int][]int) ([]string, error) {
	var result []string
	for _, version := range versions {
		major, exists := supportedVersions[int(version.Major)]
		if !exists {
			continue
		}

		if contains(major, int(version.Minor)) {
			result = append(result, version.String())
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("Trying to return a empty patch list")
	}

	return result, nil
}

func contains(inputList []int, value int) bool {
	for _, i := range inputList {
		if i == value {
			return true
		}
	}
	return false
}

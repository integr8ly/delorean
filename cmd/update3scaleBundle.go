package cmd

import (
	"errors"
	"io/fs"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	bundleName     string
	bundleImg      string
	bundleFilePath string
)

var update3scaleBundleCmd = &cobra.Command{
	Use:   "update-3scale-bundle",
	Short: "Update 3scale bundles file",
	Long: `Updates the 3scale bundle file with the information provided.

Example Usage:

# Update the bundle block in the integreatly-operator 'bundle.yaml' file.
export BUNDLE_NAME=<replace_me_with_valid_bundle_name>
export BUNDLE_IMAGE=<replace_me_with_bundle_image>
export BUNDLE_FILE=../integreatly-operator/bundles/3scale-operator/bundles.yaml
./delorean ews update-3scale-bundle --name $BUNDLE_NAME --bundle $BUNDLE_IMAGE --bundle-file $BUNDLE_FILE`,
	Run: func(cmd *cobra.Command, args []string) {
		command := &Update3scaleBundleCommand{
			BundleName:     bundleName,
			BundleImage:    bundleImg,
			BundleFilePath: bundleFilePath,
		}
		if err := command.Run(); err != nil {
			handleError(err)
		}
	},
}

type Update3scaleBundleCommand struct {
	BundleName     string
	BundleImage    string
	BundleFilePath string
}

type BundleList struct {
	Bundles []*Bundle `yaml:"bundles"`
}

type Bundle struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}

func (cmd *Update3scaleBundleCommand) Run() error {
	bundles, err := cmd.getBundles()
	if err != nil {
		return err
	}

	if err = cmd.addBundle(bundles); err != nil {
		return err
	}

	return cmd.saveBundles(bundles)
}

func (cmd *Update3scaleBundleCommand) getBundles() (*BundleList, error) {
	file, err := os.ReadFile(cmd.BundleFilePath)
	if err != nil {
		return nil, err
	}

	bundles := &BundleList{}
	err = yaml.Unmarshal(file, bundles)
	return bundles, err
}

func (cmd *Update3scaleBundleCommand) saveBundles(bl *BundleList) error {
	out, err := yaml.Marshal(bl)
	if err != nil {
		return err
	}

	return os.WriteFile(cmd.BundleFilePath, out, fs.ModeAppend)
}

func (cmd *Update3scaleBundleCommand) addBundle(bl *BundleList) error {
	for _, bundle := range bl.Bundles {
		if bundle.Name == cmd.BundleName || bundle.Image == cmd.BundleImage {
			return errors.New("Bundle or image already exists in file")
		}
	}

	newBundle := &Bundle{}
	if cmd.BundleName == "" {
		return errors.New("Bundle name cannot be an empty string")
	}
	newBundle.Name = cmd.BundleName

	if cmd.BundleImage == "" {
		return errors.New("Bundle image cannot be an empty string")
	}
	newBundle.Image = cmd.BundleImage

	bl.Bundles = append(bl.Bundles, newBundle)

	sort.Slice(bl.Bundles, func(i, j int) bool {
		return bl.Bundles[i].Name < bl.Bundles[j].Name
	})
	return nil
}

func init() {
	ewsCmd.AddCommand(update3scaleBundleCmd)

	update3scaleBundleCmd.Flags().StringVarP(
		&bundleName,
		"name",
		"n",
		"",
		"Bundle image to be included",
	)
	update3scaleBundleCmd.MarkFlagRequired("name")

	update3scaleBundleCmd.Flags().StringVarP(
		&bundleImg,
		"bundle",
		"b",
		"",
		"Bundle image to be included",
	)
	update3scaleBundleCmd.MarkFlagRequired("bundle")

	update3scaleBundleCmd.Flags().StringVarP(
		&bundleFilePath,
		"bundle-file",
		"f",
		"bundles/3scale-operator/bundles.yaml",
		"Path to the bundle.yaml file to update with the new bundle",
	)
}

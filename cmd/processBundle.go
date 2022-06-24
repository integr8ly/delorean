package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	indexImage               string
	bundleImage              string
	productsInstallationPath string
	productKey               string
	channel                  string
	bundleDir                string
	indexPackage             string
)

var (
	bundleInstallFrom = "implicit"
	indexInstallFrom  = "index"
)

var processBundleCmd = &cobra.Command{
	Use:   "process-bundle",
	Short: "Process OLM bundle image",
	Long: `Process a given OLM bundle image to be included in the products installation.

Example Usage:

# Update the 3scale product's block in the integreatly-operator 'installation.yaml' file.
export BUNDLE_IMAGE=<replace_me_with_valid_bundle>
export BUNDLE_DIR=<replace_me_with_valid_directory>
export PRODUCTS_PATH=../integreatly-operator/products/installation-cpaas.yaml
export PRODUCT_NAME=3scale
export CHANNEL=threescale-2.11
./delorean ews process-bundle --bundle $BUNDLE_IMAGE --bundle-dir $BUNDLE_DIR --products-path $PRODUCTS_PATH --product $PRODUCT --channel $CHANNEL`,
	Run: func(cmd *cobra.Command, args []string) {
		if (bundleImage == "" && indexImage == "") || (bundleImage != "" && indexImage != "") {
			handleError(errors.New("must provide either --index or --bundle"))
		}
		updater := &ProductInstallationCompositeUpdater{}
		if bundleImage != "" {
			updater.Updaters = []ProductInstallationUpdater{
				&ProductInstallationUpdaterFromValues{
					Bundle:      &bundleImage,
					InstallFrom: &bundleInstallFrom,
				},
			}
		} else {
			updater.Updaters = []ProductInstallationUpdater{
				&ProductInstallationUpdaterFromValues{
					Index:       &indexImage,
					InstallFrom: &indexInstallFrom,
					Package:     &indexPackage,
					Channel:     &channel,
				},
			}
		}
		if bundleDir != "" {
			updater.Updaters = append(updater.Updaters, &ProductInstallationUpdaterFromBundle{
				BundleDir: bundleDir,
			})
		}

		command := &ProcessBundleCommand{
			ProductsInstallationPath: productsInstallationPath,
			ProductKey:               productKey,
			Updater:                  updater,
		}
		if err := command.Run(); err != nil {
			handleError(err)
		}
	},
}

type ProcessBundleCommand struct {
	ProductsInstallationPath,
	ProductKey string
	Updater ProductInstallationUpdater
}

type ProductInstallationUpdater interface {
	UpdateProductInstallation(*ProductInstallation) error
}

type ProductInstallationUpdaterFromBundle struct {
	BundleDir string
}

var _ ProductInstallationUpdater = &ProductInstallationUpdaterFromBundle{}

func (u *ProductInstallationUpdaterFromBundle) UpdateProductInstallation(p *ProductInstallation) error {
	annotationsPath := path.Join(u.BundleDir, "metadata", "annotations.yaml")
	if _, err := os.Stat(annotationsPath); errors.Is(err, os.ErrNotExist) {
		return err
	}

	annotationsContent, err := os.ReadFile(annotationsPath)
	if err != nil {
		return err
	}

	annotations := &manifests.AnnotationsFile{}
	if err := yaml.Unmarshal(annotationsContent, annotations); err != nil {
		return err
	}

	p.Channel = annotations.Annotations.DefaultChannelName
	p.Package = annotations.Annotations.PackageName

	return nil
}

type ProductInstallationUpdaterFromValues struct {
	Channel      *string
	Bundle       *string
	InstallFrom  *string
	ManifestsDir *string
	Package      *string
	Index        *string
}

func (u *ProductInstallationUpdaterFromValues) UpdateProductInstallation(p *ProductInstallation) error {
	if p == nil {
		return errors.New("the product doesn't exist in the parsed ProductInstallation")
	}

	if u.Channel != nil {
		if *u.Channel != "" {
			p.Channel = *u.Channel
		}
	}
	if u.Bundle != nil {
		if *u.Bundle == "" {
			return errors.New("the bundle image must be specified and can't be an empty string")
		}
		p.Bundle = *u.Bundle
	}
	if u.InstallFrom != nil {
		p.InstallFrom = *u.InstallFrom
	}
	if u.ManifestsDir != nil {
		p.ManifestsDir = u.ManifestsDir
	}
	if u.Package != nil {
		p.Package = *u.Package
	}
	if u.Index != nil {
		p.Index = *u.Index
	}

	return nil
}

type ProductInstallationCompositeUpdater struct {
	Updaters []ProductInstallationUpdater
}

func (u *ProductInstallationCompositeUpdater) UpdateProductInstallation(p *ProductInstallation) error {
	for _, updater := range u.Updaters {
		if err := updater.UpdateProductInstallation(p); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *ProcessBundleCommand) Run() error {
	in, err := cmd.getProductsInstallation()
	if err != nil {
		return err
	}

	productsInstallation := &ProductsInstallation{}
	comment := ""

	if in.Content[0].Content[0].Kind == 8 {
		comment = in.Content[0].Content[0].HeadComment
	}

	err = in.Content[0].Decode(productsInstallation)
	if err != nil {
		fmt.Println("Failed to decode")
	}

	if err := cmd.Updater.UpdateProductInstallation(productsInstallation.Products[cmd.ProductKey]); err != nil {
		return err
	}

	in.Content[0].Encode(productsInstallation)
	in.Content[0].HeadComment = comment

	return cmd.saveProductsInstallation(in)
}

type ProductsInstallation struct {
	Products map[string]*ProductInstallation `yaml:"products"`
}

type ProductInstallation struct {
	Channel      string  `yaml:"channel"`
	Bundle       string  `yaml:"bundle,omitempty"`
	InstallFrom  string  `yaml:"installFrom"`
	ManifestsDir *string `yaml:"manifestsDir,omitempty"`
	Package      string  `yaml:"package,omitempty"`
	Index        string  `yaml:"index,omitempty"`
}

func (cmd *ProcessBundleCommand) getProductsInstallation() (*yaml.Node, error) {
	file, err := ioutil.ReadFile(cmd.ProductsInstallationPath)
	if err != nil {
		return nil, err
	}

	var in yaml.Node
	err = yaml.Unmarshal(file, &in)
	return &in, err
}

func (cmd *ProcessBundleCommand) saveProductsInstallation(i *yaml.Node) error {
	out, err := yaml.Marshal(i)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cmd.ProductsInstallationPath, out, fs.ModeAppend)
}

func init() {
	ewsCmd.AddCommand(processBundleCmd)

	processBundleCmd.Flags().StringVarP(
		&bundleImage,
		"bundle",
		"b",
		"",
		"Bundle image to be included",
	)

	processBundleCmd.Flags().StringVarP(
		&indexImage,
		"index",
		"i",
		"",
		"Index image to be included",
	)

	processBundleCmd.Flags().StringVarP(
		&indexPackage,
		"package",
		"o",
		"",
		"Index image to be included",
	)

	processBundleCmd.Flags().StringVarP(
		&bundleDir,
		"bundle-dir",
		"d",
		"",
		"Directory containing the bundle. If present, the `channel` and `package` fields will be populated from the bundle annotations",
	)

	processBundleCmd.Flags().StringVarP(
		&productsInstallationPath,
		"products-path",
		"f",
		"products/installation-cpaas.yaml",
		"Path to the installation.yaml file to update with the bundle",
	)

	processBundleCmd.Flags().StringVarP(
		&productKey,
		"product",
		"p",
		"",
		"Name of the product to update. It will be reflected as a key in the products object of the specified installation.yaml file",
	)
	processBundleCmd.MarkFlagRequired("product")

	processBundleCmd.Flags().StringVarP(
		&channel,
		"channel",
		"c",
		"",
		"Channel where the operator is delivered in the bundle",
	)
}

package cmd

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	bundleImage              string
	productsInstallationPath string
	productKey               string
	channel                  string
	bundleDir                string
)

var (
	bundleInstallFrom = "implicit"
)

var processBundleCmd = &cobra.Command{
	Use:   "process-bundle",
	Short: "Process a given bundle image to be included in the products installation.",
	Long:  "Process a given bundle image to be included in the products installation.",
	Run: func(cmd *cobra.Command, args []string) {
		updater := &ProductInstallationCompositeUpdater{
			Updaters: []ProductInstallationUpdater{
				&ProductInstallationUpdaterFromValues{
					Bundle:      &bundleImage,
					InstallFrom: &bundleInstallFrom,
				},
			},
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
	if u.Channel != nil {
		p.Channel = *u.Channel
	}
	if u.Bundle != nil {
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
	productsInstallation, err := cmd.getProductsInstallation()
	if err != nil {
		return err
	}

	if err := cmd.Updater.UpdateProductInstallation(productsInstallation.Products[productKey]); err != nil {
		return err
	}

	return cmd.saveProductsInstallation(productsInstallation)
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

func (cmd *ProcessBundleCommand) getProductsInstallation() (*ProductsInstallation, error) {
	file, err := ioutil.ReadFile(cmd.ProductsInstallationPath)
	if err != nil {
		return nil, err
	}

	result := &ProductsInstallation{}
	err = yaml.Unmarshal(file, result)
	return result, err
}

func (cmd *ProcessBundleCommand) saveProductsInstallation(i *ProductsInstallation) error {
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
	processBundleCmd.MarkFlagRequired("bundle")

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
		"Path to the installation.yaml file to update with the bundle.",
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
		"alpha",
		"Channel where the operator is delivered in the bundle",
	)
}

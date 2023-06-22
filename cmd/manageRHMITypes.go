package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type manageTypesCmdOptions struct {
	filepath        string
	product         string
	operatorVersion string
	productVersion  string
}

const (
	OperatorVersionType = "OperatorVersion"
	ProductVersionType  = "Version"
)

func init() {

	flags := &manageTypesCmdOptions{}

	cmd := &cobra.Command{
		Use:   "set-product-operator-version",
		Short: "Sets the operator version for a product in the rhmi_types file and sets product version to product-version if supplied",
		Run: func(cmd *cobra.Command, args []string) {
			err := SetVersion(flags.filepath, flags.product, flags.operatorVersion, flags.productVersion)
			if err != nil {
				handleError(err)
			}
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVarP(&flags.filepath, "filepath", "f", "", "Path to rhmi_types file")
	cmd.MarkFlagRequired("filepath")

	cmd.Flags().StringVarP(&flags.product, "product", "p", "", "The product name")
	cmd.MarkFlagRequired("product")

	cmd.Flags().StringVarP(&flags.operatorVersion, "version", "v", "", "The desired operator version")
	cmd.MarkFlagRequired("version")

	cmd.Flags().StringVarP(&flags.productVersion, "product-version", "e", "", "The desired product version")
}

func SetVersion(filepath string, product string, operatorVersion string, productVersion string) error {
	product = PrepareProductName(product)
	fmt.Printf("setting version of operator %s to %s\n", product, operatorVersion)
	read, err := os.Open(filepath)
	if err != nil {
		return err
	}
	bytes, err := io.ReadAll(read)
	if err != nil {
		return err
	}
	out, err := ParseVersion(string(bytes), product, operatorVersion, OperatorVersionType)
	if err != nil {
		fmt.Printf("error: %s not writing to file\n", err)
		return nil
	}
	if productVersion != "" {
		fmt.Printf("setting version of product %s to %s\n", product, productVersion)
		if out != "" {
			out, err = ParseVersion(out, product, productVersion, ProductVersionType)
		} else {
			out, err = ParseVersion(string(bytes), product, productVersion, ProductVersionType)
		}
		if err != nil {
			fmt.Printf("error: %s not writing to file\n", err)
			return nil
		}
	}
	if out != "" {
		fmt.Printf("writing changes to rhmi_types file at %s\n", filepath)
		err = os.WriteFile(filepath, []byte(out), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func PrepareProductName(product string) string {
	switch product {
	case "3scale":
		product = "3Scale"
	case "amq-online":
		product = "AMQOnline"
	case "amq-streams":
		product = "AMQStreams"
	case "apicurito":
		product = "Apicurito"
	case "codeready-workspaces":
		product = "CodeReadyWorkspaces"
	case "fuse-online":
		product = "FuseOnline"
	case "rhsso":
		product = "RHSSO"
	case "rhssouser":
		product = "RHSSOUser"
	}

	return product
}

func ParseVersion(input string, product string, version string, versionType string) (string, error) {
	// remove whitespace and "'s
	version = strings.ReplaceAll(version, "\"", "")
	version = strings.TrimSpace(version)

	var ReVersion = regexp.MustCompile(versionType + product + `.*`)

	foundVersion := ReVersion.FindString(input)

	// remove whitespace and "'s
	vs := strings.Split(foundVersion, "=")[1]
	vs = strings.ReplaceAll(vs, "\"", "")
	vs = strings.TrimSpace(vs)

	currentVersion := "v" + vs
	newVersion := "v" + version

	var out string
	fmt.Printf("current %s: %s Supplied version: %s\n", versionType, currentVersion, newVersion)
	if !semver.IsValid(currentVersion) || !semver.IsValid(newVersion) {
		return "", fmt.Errorf("one of the versions provided are invalid semver")
	}
	c := semver.Compare(currentVersion, newVersion)
	// the operator version is less than the new version
	switch c {
	case -1:
		r := strings.Replace(foundVersion, vs, version, 1)
		out = strings.Replace(input, foundVersion, r, 1)
		return out, nil
	case 0:
		fmt.Printf("%ss match or invalid, not updating types file\n", versionType)
		return "", nil
	case 1:
		return "", fmt.Errorf("current %s %s is greater than supplied version %s", versionType, currentVersion, newVersion)
	}
	return "", fmt.Errorf("unexpected operator version found")
}

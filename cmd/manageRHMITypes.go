package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type manageTypesCmdOptions struct {
	filepath string
	product  string
	version  string
}

func init() {

	flags := &manageTypesCmdOptions{}

	cmd := &cobra.Command{
		Use:   "set-product-operator-version",
		Short: "Sets the operator version for a product in the rhmi_types file and sets product to CHANGEME if a minor version update",
		Run: func(cmd *cobra.Command, args []string) {
			err := SetVersion(flags.filepath, flags.product, flags.version)
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

	cmd.Flags().StringVarP(&flags.version, "version", "v", "", "The desired version")
	cmd.MarkFlagRequired("version")
}

func SetVersion(filepath string, product string, version string) error {
	product = PrepareProductName(product)
	fmt.Printf("setting version of product operator %s to %s\n", product, version)
	read, err := os.Open(filepath)
	if err != nil {
		return err
	}
	bytes, err := ioutil.ReadAll(read)
	if err != nil {
		return err
	}
	out, err := ParseVersion(bytes, product, version)
	if err != nil {
		fmt.Printf("error: %s not writing to file\n", err)
		return nil
	}
	if out != "" {
		fmt.Printf("writing changes to rhmi_types file at %s\n", filepath)
		err = ioutil.WriteFile(filepath, []byte(out), 0644)
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
	}

	return product
}

func ParseVersion(in []byte, product string, version string) (string, error) {
	// remove whitespace and "'s
	version = strings.ReplaceAll(version, "\"", "")
	version = strings.TrimSpace(version)

	var ReOperatorVersion = regexp.MustCompile(`OperatorVersion` + product + `.*`)

	operatorVersion := ReOperatorVersion.FindString(string(in))

	// remove whitespace and "'s
	ovs := strings.Split(operatorVersion, "=")[1]
	ovs = strings.ReplaceAll(ovs, "\"", "")
	ovs = strings.TrimSpace(ovs)

	currentOperatorVersion := "v" + ovs
	newOperatorVersion := "v" + version

	var out string
	fmt.Printf("current OperatorVersion: %s Supplied OperatorVersion: %s\n", currentOperatorVersion, newOperatorVersion)
	if !semver.IsValid(currentOperatorVersion) || !semver.IsValid(newOperatorVersion) {
		return "", fmt.Errorf("one of the versions provided are invalid semver")
	}
	c := semver.Compare(currentOperatorVersion, newOperatorVersion)
	// the operator version is less than the new version
	switch c {
	case -1:
		r := strings.Replace(operatorVersion, ovs, version, 1)
		out = strings.Replace(string(in), operatorVersion, r, 1)
		return out, nil
	case 0:
		fmt.Println("Operator versions match or invalid, not updating types file")
		return "", nil
	case 1:
		return "", fmt.Errorf("current operator version %s is greater than supplied version %s", currentOperatorVersion, newOperatorVersion)
	}
	return "", fmt.Errorf("unexpected operator version found")
}

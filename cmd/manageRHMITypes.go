package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
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
	fmt.Println(fmt.Sprintf("setting version of product operator %s to %s", product, version))
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
		fmt.Println(fmt.Sprintf("error: %s not writing to file", err))
		return nil
	}
	fmt.Println(fmt.Sprintf("writing changes to rhmi_types file at %s ", filepath))
	err = ioutil.WriteFile(filepath, []byte(out), 600)
	if err != nil {
		return err
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

	currentOperatorMajor := semver.MajorMinor("v" + ovs)
	newOperatorMajor := semver.MajorMinor("v" + version)

	var out string
	fmt.Println(fmt.Sprintf("current OperatorVersion: %s Supplied OperatorVersion: %s", currentOperatorMajor, newOperatorMajor))
	c := semver.Compare(currentOperatorMajor, newOperatorMajor)
	// major versions are a match
	if c == 0 || c == -1 || currentOperatorMajor == "" && newOperatorMajor == "" {
		r := strings.Replace(operatorVersion, ovs, version, 1)
		out = strings.Replace(string(in), operatorVersion, r, 1)
		return out, nil
	}

	if c == 1 {
		return "", fmt.Errorf(fmt.Sprintf("current operator version %s is greater than supplied version %s", currentOperatorMajor, newOperatorMajor))
	}

	return "", fmt.Errorf("unexpected operator version found")
}

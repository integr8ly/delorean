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
	product   string
	version   string
}

func init() {

	flags := &manageTypesCmdOptions{}

	cmd := &cobra.Command{
		Use:   "set-product-version",
		Short: "Sets the operator and product version for a product in the rhmi_types file",
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
	fmt.Println(fmt.Sprintf("setting version of product %s to %s", product, version))
	read, err := os.Open(filepath)
	if err != nil {
		return err
	}
	bytes, err := ioutil.ReadAll(read)
	if err != nil {
		return err
	}
	out := ParseVersion(bytes, product, version)
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

func ParseVersion(in []byte, product string, version string) string {
	// remove whitespace and "'s
	version = strings.ReplaceAll(version, "\"", "")
	version = strings.TrimSpace(version)

	var ReOperatorVersion = regexp.MustCompile(`OperatorVersion` + product + `.*`)
	var ReProductVersion = regexp.MustCompile(`Version` + product + `.*`)

	operatorVersion := ReOperatorVersion.FindString(string(in))
	productVersion := ReProductVersion.FindString(string(in))

	// remove whitespace and "'s
	ovs := strings.Split(operatorVersion, "=")[1]
	ovs = strings.ReplaceAll(ovs, "\"", "")
	ovs = strings.TrimSpace(ovs)

	pvs := strings.Split(productVersion, "=")[1]

	currentOperatorMajor := semver.Major("v" + ovs)
	newOperatorMajor := semver.Major("v" + version)

	var out string
	// we only want to set the Version<product> var to CHANGEME for minor releases
	fmt.Println(fmt.Sprintf("current OperatorVersion: %s Found OperatorVersion: %s", currentOperatorMajor, newOperatorMajor))
	c := semver.Compare(currentOperatorMajor, newOperatorMajor)
	// major versions are a match
	if c == 0 {
		out = strings.Replace(string(in), pvs, " \"CHANGEME\"", 1)
		r := strings.Replace(operatorVersion, ovs, version, 1)
		out = strings.Replace(out, operatorVersion, r, 1)
	}

	// if it's not a minor release only change the OperatorVersion<product> var
	// if the version is 0.x.x then semver returns empty string so account for it
	if c != 0 || currentOperatorMajor == "" && newOperatorMajor == "" {
		out = strings.Replace(string(in), ovs, version, 1)
	}

	return out
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// extractManifestsCmd represents the extractManifests command
var extractManifestsCmd = &cobra.Command{
	Use:   "extract-manifests",
	Short: "Extract OLM Manifest bundle",
	Long:  `Extracts any olm manifest bundles contained within a given container image and outputs it, unmodified, to a given directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("extractManifests called")
	},
}

func init() {
	ewsCmd.AddCommand(extractManifestsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// extractManifestsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// extractManifestsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

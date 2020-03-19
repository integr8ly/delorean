package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// processImageManifestsCmd represents the processImageManifests command
var processImageManifestsCmd = &cobra.Command{
	Use:   "processImageManifests",
	Short: "Extracts a manifest bundle from a given container image",
	Long: `Extracts the most recent manifests from a given container image containing
a manifest bundle and processes it into a version that will work with the target operator e.g. integreatly-operator`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("processImageManifests called")
	},
}

func init() {
	rootCmd.AddCommand(processImageManifestsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// processImageManifestsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// processImageManifestsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// processCSVImagesCmd represents the processCSVImages command
var processCSVImagesCmd = &cobra.Command{
	Use:   "process-csv-images",
	Short: "Replace internal image registry references and generates an image mirror mapping file.",
	Long: `Locates the current cluster service version file (csv) for a given product and replaces all occurrences of 
internal image registries with a deloeran version and generates an image_mirror_mapping file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("processCSVImages called")
	},
}

func init() {
	ewsCmd.AddCommand(processCSVImagesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// processCSVImagesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// processCSVImagesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

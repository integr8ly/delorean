package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// mirrorImagesCmd represents the mirrorImages command
var mirrorImagesCmd = &cobra.Command{
	Use:   "mirrorImages",
	Short: "Mirror images defined in image mirror mapping files.",
	Long: `Iterate through all image mappings found in any image_mirror_mapping files
inside each products manifest directory and mirrors the images defined.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("mirrorImages called")
	},
}

func init() {
	rootCmd.AddCommand(mirrorImagesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mirrorImagesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mirrorImagesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

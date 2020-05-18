package cmd

import (
	"fmt"

	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type currentCSVFlags struct {
	directory string
	output    string
}

func init() {

	flags := &currentCSVFlags{}

	cmd := &cobra.Command{
		Use:   "current-csv",
		Short: "Retrieve the current CSV from the manifests directory and write it in JSON format to the output file",
		Run: func(cmd *cobra.Command, args []string) {
			csv, file, err := utils.GetCurrentCSV(flags.directory)
			if err != nil {
				handleError(err)
			}

			fmt.Printf("Write current CSV %s to %s\n", file, flags.output)
			err = csv.WriteJSON(flags.output)
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVarP(&flags.directory, "directory", "d", "", "Path to the directory containing the manifests from which to extract the current CSV")
	cmd.MarkFlagRequired("directory")

	cmd.Flags().StringVarP(&flags.output, "output", "o", "", "File path in which to write the current CSV in JSON")
	cmd.MarkFlagRequired("output")
}

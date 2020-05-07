package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
			bytes, err := json.Marshal(csv)
			if err != nil {
				handleError(err)
			}

			// truncate the existing file
			write, err := os.Create(flags.output)
			if err != nil {
				handleError(err)
			}

			_, err = write.Write(bytes)
			if err != nil {
				handleError(err)
			}

			err = write.Close()
			if err != nil {
				handleError(err)
			}
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVarP(&flags.directory, "directory", "d", "", "Path to the directory containing the manifests from which to extract the current CSV")
	cmd.MarkFlagRequired("directory")

	cmd.Flags().StringVarP(&flags.output, "output", "o", "", "File path in which to write the current CSV in JSON")
	cmd.MarkFlagRequired("output")
}

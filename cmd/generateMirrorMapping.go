package cmd

import (
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"

	"errors"
	"github.com/spf13/cobra"
)

type mirrorImageFlags struct {
	imageType string
	newImage  string
	directory string
}

func init() {
	flags := &mirrorImageFlags{}

	cmd := &cobra.Command{
		Use:   "generate-mirror-mapping",
		Short: "Generate a mirror mapping file specifying src to dst",
		Run: func(cmd *cobra.Command, args []string) {
			err := generateMirrorMapping(flags)
			if err != nil {
				handleError(err)
			}
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVar(&flags.imageType, "imageType", "", "the type of image to mpa")
	cmd.MarkFlagRequired("imageType")

	cmd.Flags().StringVar(&flags.newImage, "newImage", "", "the new image to map")
	cmd.MarkFlagRequired("newImage")

	cmd.Flags().StringVar(&flags.directory, "directory", "", "directory to write mapping file")
	cmd.MarkFlagRequired("directory")
}

func generateMirrorMapping(flags *mirrorImageFlags) error {
	imageType := flags.imageType
	newImage := flags.newImage
	directory := flags.directory

	if !utils.IsValidType(imageType) {
		return errors.New(fmt.Sprintf("Invalid image type %s", imageType))
	}

	return utils.CreateMirrorMap(directory, imageType, newImage)
}

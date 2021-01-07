package cmd

import (
	"errors"
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type replaceImageFlags struct {
	imageType string
	newImage  string
	opDir     string
}

func init() {

	flags := &replaceImageFlags{}

	cmd := &cobra.Command{
		Use:   "replace-image-version",
		Short: "Replace existing image reference in integreatly operator with new image",
		Run: func(cmd *cobra.Command, args []string) {
			err := replaceImage(flags)
			if err != nil {
				handleError(err)
			}
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVar(&flags.imageType, "imageType", "", "the type of image being replaced")
	cmd.MarkFlagRequired("imageType")

	cmd.Flags().StringVar(&flags.newImage, "newImage", "", "the new image to update to")
	cmd.MarkFlagRequired("newImage")

	cmd.Flags().StringVar(&flags.opDir, "opDir", "", "the intergreatly operator directory")
	cmd.MarkFlagRequired("opDir")
}

func replaceImage(flags *replaceImageFlags) error {
	imageType := flags.imageType
	opDir := flags.opDir
	newImage := flags.newImage

	if !utils.IsValidType(imageType) {
		return errors.New(fmt.Sprintf("Invalid image type %s", imageType))
	}

	imageDtls := utils.ImageSubs[imageType]
	err := imageDtls.ReplaceImage(opDir, imageDtls.FileLocation(opDir), imageDtls.LineRegEx, newImage)
	if err != nil {
		return errors.New(fmt.Sprintf("Error replacing image. newImage %s. Error: %v", newImage, err))
	}
	fmt.Println(fmt.Sprintf("Successfully replaced image"))
	return nil
}

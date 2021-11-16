package cmd

import (
	"errors"
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type replaceImageFlags struct {
	imageType   string
	newImageTag string
	opDir       string
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

	cmd.Flags().StringVar(&flags.newImageTag, "newImageTag", "", "the new image tag to update to")
	cmd.MarkFlagRequired("newImageTag")

	cmd.Flags().StringVar(&flags.opDir, "opDir", "", "the intergreatly operator directory")
	cmd.MarkFlagRequired("opDir")
}

func replaceImage(flags *replaceImageFlags) error {
	imageType := flags.imageType
	opDir := flags.opDir
	newImageTag := flags.newImageTag

	if !utils.IsValidType(imageType) {
		return errors.New(fmt.Sprintf("Invalid image type %s", imageType))
	}

	imageDtls := utils.ImageSubs[imageType]
	newImage := utils.ImageSubs[imageType].MirrorRepo + ":" + newImageTag
	err := imageDtls.ReplaceImage(opDir, imageDtls.FileLocation(opDir), imageDtls.LineRegEx, newImage)
	if err != nil {
		return errors.New(fmt.Sprintf("Error replacing image. newImageTag %s. Error: %v", newImageTag, err))
	}
	fmt.Println(fmt.Sprintf("Successfully replaced image"))
	return nil
}

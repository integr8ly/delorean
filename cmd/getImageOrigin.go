package cmd

import (
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"

	"errors"
	"github.com/spf13/cobra"
)

type confirmImageOriginFlags struct {
	imageType string
	imageTag  string
}

func init() {
	flags := &confirmImageOriginFlags{}

	cmd := &cobra.Command{
		Use:   "get-image-origin",
		Short: "Based on imageType and tag, return the image origin url if it exists",
		Run: func(cmd *cobra.Command, args []string) {
			imageUrl, err := confirmImageOrigin(flags)
			if err != nil {
				handleError(err)
				return
			}
			fmt.Println(imageUrl)
		},
	}

	ewsCmd.AddCommand(cmd)

	cmd.Flags().StringVar(&flags.imageType, "imageType", "", "the type of image to check")
	cmd.MarkFlagRequired("imageType")

	cmd.Flags().StringVar(&flags.imageTag, "imageTag", "", "the image tag to check")
	cmd.MarkFlagRequired("imageTag")
}

func confirmImageOrigin(flags *confirmImageOriginFlags) (string, error) {
	imageType := flags.imageType
	imageTag := flags.imageTag

	if !utils.IsValidType(imageType) {
		return "", errors.New(fmt.Sprintf("Invalid image type %s", imageType))
	}

	imageDtls := utils.ImageSubs[imageType]
	return imageDtls.GetOriginImage(imageDtls.OriginRepo, imageTag)
}

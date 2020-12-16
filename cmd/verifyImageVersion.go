package cmd

import (
	"errors"
	"fmt"
	"github.com/blang/semver"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type verifyImageVersionFlags struct {
	imageType string
	newImage  string
	opDir     string
}

func init() {

	flags := &verifyImageVersionFlags{}

	cmd := &cobra.Command{
		Use:   "verify-image-version",
		Short: "Verify that a new image version is ahead of the current image version in the integreatly operator",
		Run: func(cmd *cobra.Command, args []string) {
			err := verifyImageVersion(flags)
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

func verifyImageVersion(flags *verifyImageVersionFlags) error {
	imageType := flags.imageType
	opDir := flags.opDir
	newImage := flags.newImage

	if !utils.IsValidType(imageType) {
		return errors.New(fmt.Sprintf("Invalid image type %s", imageType))
	}

	imageDtls := utils.ImageSubs[imageType]
	currentVersion, err := imageDtls.GetCurrentVersion(opDir, imageDtls.FileLocation(opDir), imageDtls.LineRegEx)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting current version of imageType %s. Error: %v", imageType, err))
	}
	newVersion, err := utils.GetNewVersion(newImage)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting new version of imageType %s. Error: %v", imageType, err))
	}
	newer, err := isNewerVersion(currentVersion, newVersion)
	if err != nil {
		return err
	}
	if newer {
		fmt.Println(fmt.Sprintf("New version %s, is a valid update", newImage))
		return nil
	}
	return errors.New(fmt.Sprintf("The new image is not ahead of the current. NewImage: %s. CurrentVersion: %s", newImage, currentVersion))
}

func isNewerVersion(current *semver.Version, new *semver.Version) (bool, error) {
	if new.Compare(*current) == 1 {
		return true, nil
	}
	return false, nil
}

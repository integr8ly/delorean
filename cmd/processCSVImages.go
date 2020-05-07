package cmd

import (
	"context"
	"errors"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"path"
)

type processCSVImagesCmdOptions struct {
	manifestDir string
	isGa        bool
	extraImages []string
}

type ProcessCSVImagesCmd func(manifestDir string, isGa bool, extraImages string) error

var processCSVImagesCmdOpts = &processCSVImagesCmdOptions{}

// processCSVImagesCmd represents the processCSVImages command
var processCSVImagesCmd = &cobra.Command{
	Use:   "process-csv-images",
	Short: "Replace internal image registry references and generates an image mirror mapping file.",
	Long: `Locates the current cluster service version file (csv) for a given product and replaces all occurrences of 
internal image registries with a delorean version and generates an image_mirror_mapping file.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := DoProcessCSV(cmd.Context(), processCSVImagesCmdOpts)
		if err != nil {
			handleError(err)
		}

	},
}

func DoProcessCSV(ctx context.Context, cmdOpts *processCSVImagesCmdOptions) error {
	if cmdOpts.manifestDir == "" {
		return errors.New("manifest-dir not specified")
	}

	//verify it's a manifest dir.
	err := utils.VerifyManifestDirs(cmdOpts.manifestDir)
	if err != nil {
		handleError(err)
	}

	if !cmdOpts.isGa {
		images, err := utils.GetAndUpdateOperandImagesToDeloreanImages(cmdOpts.manifestDir, cmdOpts.extraImages)
		images, err = utils.UpdateOperatorImagesToDeloreanImages(cmdOpts.manifestDir, images)
		if err != nil {
			handleError(err)
		}
		if len(images) > 0 {
			err = utils.CreateImageMirrorMappingFile(cmdOpts.manifestDir, images)
			if err != nil {
				handleError(err)
			}
		}
	}

	if cmdOpts.isGa {
		if utils.FileExists(path.Join(cmdOpts.manifestDir, utils.MappingFile)) {
			err := os.Remove(path.Join(cmdOpts.manifestDir, utils.MappingFile))
			if err != nil {
				handleError(err)
			}
		}
	}

	return nil
}

func init() {
	ewsCmd.AddCommand(processCSVImagesCmd)

	processCSVImagesCmd.Flags().StringVarP(&processCSVImagesCmdOpts.manifestDir, "manifest-dir", "m", "", "Manifest Directory Location.")
	processCSVImagesCmd.Flags().BoolVarP(&processCSVImagesCmdOpts.isGa, "isGa", "g", false, "Mark as GA version")
	processCSVImagesCmd.Flags().StringArrayVarP(&processCSVImagesCmdOpts.extraImages, "extra-images", "e", []string{}, "Extra images to include in container env NAME@IMAGE")
}

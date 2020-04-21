package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type extractManifestsCmdOptions struct {
	srcImage string
	srcDir   string
	destDir  string
	tmpDir   string
}

var extractManifestsCmdOpts = &extractManifestsCmdOptions{}

type ExtractManifestsCmd func(image string, path string) error

func extractImageManifests(image string, destDir string) error {
	ocExecutable, err := exec.LookPath("oc")
	if err != nil {
		return err
	}

	cmdOCManifestExtract := &exec.Cmd{
		Path:   ocExecutable,
		Args:   []string{ocExecutable, "image", "extract", image, "--path", "/manifests/:" + destDir, "--confirm"},
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}

	return cmdOCManifestExtract.Run()
}

func copyLatestManifest(srcPkgDir, destPkgDir string) (string, string, error) {

	csv, srcBundleDir, err := utils.GetCurrentCSV(srcPkgDir)
	if err != nil {
		return "", "", err
	}

	destBundleDir := filepath.Join(destPkgDir, filepath.Base(srcBundleDir))
	err = utils.CopyDirectory(srcBundleDir, destBundleDir)
	if err != nil {
		return "", "", err
	}

	_, err = utils.UpdatePackageManifest(destPkgDir, csv.Name)
	if err != nil {
		return "", "", err
	}

	return srcBundleDir, destBundleDir, nil
}

func DoExtractManifests(ctx context.Context, extractManifests ExtractManifestsCmd, cmdOpts *extractManifestsCmdOptions) error {
	if cmdOpts.srcImage == "" && cmdOpts.srcDir == "" {
		return errors.New("Missing source. Must specify a source image or directory!")
	}
	if cmdOpts.srcImage != "" {
		fmt.Println("Extracting manifests from", cmdOpts.srcImage)

		err := extractManifests(cmdOpts.srcImage, cmdOpts.tmpDir)
		if err != nil {
			return err
		}
		fmt.Println("Manifests extracted to", cmdOpts.tmpDir)
		cmdOpts.srcDir = cmdOpts.tmpDir
	}

	if cmdOpts.destDir != "" {
		err := utils.VerifyManifestDirs(cmdOpts.srcDir, cmdOpts.destDir)
		if err != nil {
			return err
		}

		from, to, err := copyLatestManifest(cmdOpts.srcDir, cmdOpts.destDir)
		if err != nil {
			return err
		}
		fmt.Printf("Copied latest manifest bundle from '%s' to '%s'\n", from, to)
	}
	return nil
}

// extractManifestsCmd represents the extractManifests command
var extractManifestsCmd = &cobra.Command{
	Use:   "extract-manifests",
	Short: "Extract OLM Manifest bundle",
	Long:  `Extracts any olm manifest bundles contained within a given source container image or directory and copies the latest bundle, unmodified, to a given directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "manifests-")

		if err != nil {
			handleError(err)
		}
		extractManifestsCmdOpts.tmpDir = tmpDir
		if err := DoExtractManifests(cmd.Context(), extractImageManifests, extractManifestsCmdOpts); err != nil {
			handleError(err)
		}
	},
}

func init() {
	extractManifestsCmd.Flags().StringVarP(&extractManifestsCmdOpts.srcImage, "src-image", "i", "", "Source container image. Image must contain a /manifests directory. (Overrides 'src-dir')")
	extractManifestsCmd.Flags().StringVarP(&extractManifestsCmdOpts.srcDir, "src-dir", "s", "", "Source directory (Ignored if 'src-image' is provided).")
	extractManifestsCmd.Flags().StringVarP(&extractManifestsCmdOpts.destDir, "dest-dir", "d", "", "Destination directory. The latest bundle from the given source ('src-image' or 'src-dir') will be copied here.")
	ewsCmd.AddCommand(extractManifestsCmd)
}

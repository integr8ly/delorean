package cmd

import (
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

var manifestDir string

const (
	envVarWatchNamespace = "WATCH_NAMESPACE"
	envVarNamespace      = "NAMESPACE"
)

// processManifestCmd represents the processImageManifests command
var processManifestCmd = &cobra.Command{
	Use:   "process-manifest",
	Short: "Process a given manifest to meet the rhmi requirements.",
	Long:  `Process a given manifest to meet the rhmi requirements.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := DoProcessManifest(cmd.Context(), manifestDir)
		if err != nil {
			handleError(err)
		}
	},
}

func processManifest(csv *utils.CSV) error {
	//update "WATCH_NAMESPACE" and "NAMESPACE" env vars if present
	envKeyValMap := map[string]string{
		envVarWatchNamespace: "metadata.annotations['olm.targetNamespaces']",
		envVarNamespace:      "metadata.annotations['olm.targetNamespaces']",
	}
	err := csv.UpdateEnvVarList(envKeyValMap)
	if err != nil {
		return err
	}
	return nil
}

func DoProcessManifest(ctx context.Context, manifestDir string) error {
	//verify it's a manifest dir.
	err := utils.VerifyManifestDirs(manifestDir)
	if err != nil {
		return err
	}
	err = utils.ProcessCurrentCSV(manifestDir, processManifest)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	ewsCmd.AddCommand(processManifestCmd)

	processManifestCmd.Flags().StringVarP(&manifestDir, "manifest-dir", "m", "", "Manifest Directory Location.")
}

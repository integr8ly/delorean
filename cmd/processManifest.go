package cmd

import (
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
		//verify it's a manifest dir.
		err := utils.VerifyManifestDirs(manifestDir)
		if err != nil {
			handleError(err)
		}
		err = utils.ProcessCurrentCSV(manifestDir, processManifest)
		if err != nil {
			handleError(err)
		}
	},
}

func init() {
	ewsCmd.AddCommand(processManifestCmd)

	processManifestCmd.Flags().StringVarP(&manifestDir, "manifest-dir", "m", "", "Manifest Directory Location.")
}

func processManifest(csv *utils.CSV) error {
	//update "WATCH_NAMESPACE" and "NAMESPACE" env vars if present
	envKeyValMap := map[string]string{
		envVarWatchNamespace: "metadata.annotations['olm.targetNamespaces']",
		envVarNamespace:      "metadata.annotations['olm.targetNamespaces']",
	}
	csv.UpdateEnvVarList(envKeyValMap)

	return nil
}

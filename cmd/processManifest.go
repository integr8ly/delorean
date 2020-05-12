package cmd

import (
	"github.com/integr8ly/delorean/pkg/utils"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
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

func processManifest(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	//Get the correct replaces value and update it.
	sortedCSVs, err := utils.GetSortedCSVNames(manifestDir)
	if err != nil {
		handleError(err)
	}

	if sortedCSVs.Len() > 1 {
		csv.Spec.Replaces = sortedCSVs[(sortedCSVs.Len() - 2)].Name
	}

	//update "WATCH_NAMESPACE" and "NAMESPACE" env vars if present
	envKeyValMap := map[string]string{
		envVarWatchNamespace: "metadata.annotations['olm.targetNamespaces']",
		envVarNamespace:      "metadata.annotations['olm.targetNamespaces']",
	}
	utils.UpdateEnvVarList(csv, envKeyValMap)

	return nil
}

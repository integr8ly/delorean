package cmd

import (
	"github.com/integr8ly/delorean/pkg/utils"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
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
		//get current csv object and filepath
		csv, filepath, err := utils.GetCurrentCSV(manifestDir)
		if err != nil {
			handleError(err)
		}

		//Get the correct replaces value and update it.
		sortedCSVs, err := utils.GetSortedCSVNames(manifestDir)
		if err != nil {
			handleError(err)
		}

		if sortedCSVs.Len() > 1 {
			csv.Spec.Replaces = sortedCSVs[(sortedCSVs.Len() - 2)].Name
		}

		//update "WATCH_NAMESPACE" and "NAMESPACE" env vars if present
		updateEnvs(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs)

		//parse object to yaml and write to back to file
		err = utils.WriteObjectToYAML(csv, filepath)
		if err != nil {
			handleError(err)
		}
	},
}

func init() {
	ewsCmd.AddCommand(processManifestCmd)

	processManifestCmd.Flags().StringVarP(&manifestDir, "manifest-dir", "m", "", "Manifest Directory Location.")
}

func updateEnvs(deployments []olmapiv1alpha1.StrategyDeploymentSpec) {
	spec := deployments[0].Spec
	envs := spec.Template.Spec.Containers[0].Env
	watchNamespaceEnv := v1.EnvVar{
		Name: envVarWatchNamespace,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.annotations['olm.targetNamespaces']",
			},
		},
	}
	namespaceEnv := v1.EnvVar{
		Name: envVarNamespace,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.annotations['olm.targetNamespaces']",
			},
		},
	}

	//update "WATCH_NAMESPACE" and "NAMESPACE" env vars if present
	for i, env := range envs {
		if env.Name == envVarWatchNamespace {
			envs[i] = watchNamespaceEnv
		}
		if env.Name == envVarNamespace {
			envs[i] = namespaceEnv
		}
	}
}

package utils

import (
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func AddOrUpdateEnvVar(envVars []corev1.EnvVar, envName string, envVal string) []corev1.EnvVar {
	v := corev1.EnvVar{
		Name:  envName,
		Value: envVal,
	}
	for i, env := range envVars {
		if env.Name == envName {
			envVars[i] = v
			return envVars
		}
	}
	return append(envVars, v)
}

func UpdateEnvVarList(csv *olmapiv1alpha1.ClusterServiceVersion, envKeyValMap map[string]string) {

	for _, d := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		for _, c := range d.Spec.Template.Spec.Containers {
			for _, e := range c.Env {
				for k, v := range envKeyValMap {
					if e.Name == k {
						e.ValueFrom.FieldRef.FieldPath = v
					}
				}
			}
		}
	}

}

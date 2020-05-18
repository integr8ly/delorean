package utils

import (
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

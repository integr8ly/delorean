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

func AddOrUpdateEnvVarWithSource(envVars []corev1.EnvVar, envName string, envVal string, fieldPath string) []corev1.EnvVar {
	v := corev1.EnvVar{
		Name:  envName,
		Value: envVal,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
	for i, env := range envVars {
		if env.Name == envName {
			envVars[i] = v
			return envVars
		}
	}
	return append(envVars, v)
}

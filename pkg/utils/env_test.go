package utils

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestAddOrUpdateEnvVar(t *testing.T) {
	type args struct {
		envVars []corev1.EnvVar
		envName string
		envVal  string
	}
	tests := []struct {
		name string
		args args
		want []corev1.EnvVar
	}{
		{
			name: "test update env",
			args: args{[]corev1.EnvVar{
				{
					Name:  "TestName",
					Value: "TestVal",
				},
			}, "TestName", "TestVal2"},
			want: []corev1.EnvVar{
				{
					Name:  "TestName",
					Value: "TestVal2",
				},
			},
		},
		{
			name: "test add env",
			args: args{[]corev1.EnvVar{}, "TestName", "TestVal"},
			want: []corev1.EnvVar{
				{
					Name:  "TestName",
					Value: "TestVal",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddOrUpdateEnvVar(tt.args.envVars, tt.args.envName, tt.args.envVal); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddOrUpdateEnvVar() = %v, want %v", got, tt.want)
			}
		})
	}
}

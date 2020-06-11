package utils

import (
	"testing"
)

func TestBuildDeloreanImage(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			"test sha",
			"my.registry.io/myrepo/test-operator@sha256:1ba6ec8ed984a011796bbe1eafabb2791957f58ed66ec4a484c024dd96eaf427",
			"quay.io/integreatly/delorean:myrepo-test-operator_latest",
		},
		{
			"test tag",
			"my.registry.io/myrepo/test-operator:1.0",
			"quay.io/integreatly/delorean:myrepo-test-operator_1.0",
		},
		{
			"test no tag",
			"my.registry.io/myrepo/test-operator",
			"quay.io/integreatly/delorean:myrepo-test-operator_latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildDeloreanImage(tt.image); got != tt.want {
				t.Errorf("BuildDeloreanImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

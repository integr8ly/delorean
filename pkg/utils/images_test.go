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
			"quay.io/integreatly/delorean:myrepo-test-operator_1ba6ec8ed984a011796bbe1eafabb2791957f58ed66ec4a484c024dd96eaf427",
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
		{
			"test amq broker",
			"registry-proxy.engineering.redhat.com/rh-osbs/amq-broker-7-amq-broker-76-openshift:7.6",
			"quay.io/integreatly/delorean:rh-osbs-amq-broker-7-amq-broker-76-openshift_7.6",
		},
		{
			"test CRW operator",
			"registry-proxy.engineering.redhat.com/rh-osbs/codeready-workspaces-operator@sha256:02e8777fa295e6615bbd73f3d92911e7e7029b02cdf6346eba502aaeb8fe3de1",
			"quay.io/integreatly/delorean:rh-osbs-codeready-workspaces-operator_02e8777fa295e6615bbd73f3d92911e7e7029b02cdf6346eba502aaeb8fe3de1",
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

func TestBuildOSBSImage(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			"test sha",
			"my.registry.io/myrepo/test-operator@sha256:1ba6ec8ed984a011796bbe1eafabb2791957f58ed66ec4a484c024dd96eaf427",
			"registry-proxy.engineering.redhat.com/rh-osbs/myrepo-test-operator@sha256:1ba6ec8ed984a011796bbe1eafabb2791957f58ed66ec4a484c024dd96eaf427",
		},
		{
			"test tag",
			"my.registry.io/myrepo/test-operator:1.0",
			"registry-proxy.engineering.redhat.com/rh-osbs/myrepo-test-operator:1.0",
		},
		{
			"test no tag",
			"my.registry.io/myrepo/test-operator",
			"registry-proxy.engineering.redhat.com/rh-osbs/myrepo-test-operator",
		},
		{
			"test amq broker",
			"registry.redhat.io/amq7/amq-broker:7.6",
			"registry-proxy.engineering.redhat.com/rh-osbs/amq-broker-7-amq-broker-76-openshift:7.6",
		},
		{
			"test CRW operator",
			"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator@sha256:02e8777fa295e6615bbd73f3d92911e7e7029b02cdf6346eba502aaeb8fe3de1",
			"registry-proxy.engineering.redhat.com/rh-osbs/codeready-workspaces-operator@sha256:02e8777fa295e6615bbd73f3d92911e7e7029b02cdf6346eba502aaeb8fe3de1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildOSBSImage(tt.image); got != tt.want {
				t.Errorf("BuildOSBSImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

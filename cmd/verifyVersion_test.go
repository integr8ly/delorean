package cmd

import (
	"testing"
)

func TestVerifyVersion(t *testing.T) {
	tests := []struct {
		name    string
		flags   *verifyVersionFlags
		wantErr bool
	}{
		{
			name: "test new incoming CSV with old current CSV",
			flags: &verifyVersionFlags{
				incomingManifests: "../pkg/utils/testdata/validManifests/3scale3",
				currentManifests:  "../pkg/utils/testdata/validManifests/3scale2",
			},
			wantErr: false,
		},
		{
			name: "test equal incoming CSV with current CSV",
			flags: &verifyVersionFlags{
				incomingManifests: "../pkg/utils/testdata/validManifests/3scale",
				currentManifests:  "../pkg/utils/testdata/validManifests/3scale",
			},
			wantErr: false,
		},
		{
			name: "test old incoming CSV with newer current CSV",
			flags: &verifyVersionFlags{
				incomingManifests: "../pkg/utils/testdata/validManifests/3scale",
				currentManifests:  "../pkg/utils/testdata/validManifests/3scale2",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := doVerifyVersion(tt.flags); (err != nil) != tt.wantErr {
				t.Errorf("doVerifyVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

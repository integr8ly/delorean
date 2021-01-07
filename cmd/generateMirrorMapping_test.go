package cmd

import (
	"testing"
)

func TestMirrorImages(t *testing.T) {
	tests := []struct {
		name    string
		flags   *mirrorImageFlags
		wantErr bool
	}{
		{
			name: "test error on invalid image type",
			flags: &mirrorImageFlags{
				imageType: "invalidImage",
				newImage:  "",
				directory: "",
			},
			wantErr: true,
		},
		{
			name: "test error in invalid image format",
			flags: &mirrorImageFlags{
				imageType: "envoyProxy",
				newImage:  "invalidImage",
				directory: "",
			},
			wantErr: true,
		},
		{
			name: "test error on invalid directory to write to",
			flags: &mirrorImageFlags{
				imageType: "envoyProxy",
				newImage:  "validImage:1.1.1",
				directory: "invalidDirectory",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := generateMirrorMapping(tt.flags); (err != nil) != tt.wantErr {
				t.Errorf("verifyImageVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

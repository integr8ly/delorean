package cmd

import (
	"errors"
	"github.com/integr8ly/delorean/pkg/utils"
	"testing"
)

func getConfirmImageMock(err error) func(repo string, imageTag string) (string, error) {
	return func(repo string, imageTag string) (string, error) {
		if imageTag == "valid" {
			return "repo", nil
		}
		return "", errors.New("Error Confirm Origin Image")
	}
}

func TestConfirmImageOrigin(t *testing.T) {
	tests := []struct {
		name         string
		flags        *confirmImageOriginFlags
		getImageMock func(repo string, imageTag string) (string, error)
		wantErr      bool
	}{
		{
			name: "test replace image success",
			flags: &confirmImageOriginFlags{
				imageType: "rateLimiting",
				imageTag:  "valid",
			},
			wantErr:      false,
			getImageMock: getConfirmImageMock(nil),
		},
		{
			name: "test replace image failure",
			flags: &confirmImageOriginFlags{
				imageType: "rateLimiting",
				imageTag:  "invalid",
			},
			wantErr:      true,
			getImageMock: getConfirmImageMock(errors.New("Error confirming image")),
		},
		{
			name: "test passing invalid image type",
			flags: &confirmImageOriginFlags{
				imageType: "invalidType",
				imageTag:  "",
			},
			wantErr:      true,
			getImageMock: getConfirmImageMock(errors.New("Error confirming image")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			utils.ImageSubs["rateLimiting"].GetOriginImage = tt.getImageMock

			if _, err := confirmImageOrigin(tt.flags); (err != nil) != tt.wantErr {
				t.Errorf("verifyImageVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

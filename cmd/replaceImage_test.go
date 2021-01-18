package cmd

import (
	"errors"
	"github.com/integr8ly/delorean/pkg/utils"
	"regexp"
	"testing"
)

func getReplaceImageMock(err error) func(a string, b string, r *regexp.Regexp, c string) error {
	return func(a string, b string, r *regexp.Regexp, c string) error {
		return err
	}
}

func TestReplaceImage(t *testing.T) {
	tests := []struct {
		name             string
		flags            *replaceImageFlags
		replaceImageMock func(a string, b string, r *regexp.Regexp, c string) error
		wantErr          bool
	}{
		{
			name: "test replace image success",
			flags: &replaceImageFlags{
				opDir:       "",
				imageType:   "envoyProxy",
				newImageTag: "",
			},
			wantErr:          false,
			replaceImageMock: getReplaceImageMock(nil),
		},
		{
			name: "test replace image failure",
			flags: &replaceImageFlags{
				opDir:       "",
				imageType:   "envoyProxy",
				newImageTag: "",
			},
			wantErr:          true,
			replaceImageMock: getReplaceImageMock(errors.New("Error replacing image")),
		},
		{
			name: "test passing invalid image type",
			flags: &replaceImageFlags{
				opDir:       "",
				imageType:   "invalidType",
				newImageTag: "",
			},
			wantErr:          true,
			replaceImageMock: getReplaceImageMock(errors.New("Error replacing image")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			utils.ImageSubs["envoyProxy"].ReplaceImage = tt.replaceImageMock

			if err := replaceImage(tt.flags); (err != nil) != tt.wantErr {
				t.Errorf("verifyImageVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package cmd

import (
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
)

func verifyOLMTargetNamespace(t *testing.T, manifestDir string) error {

	_, csvFile, err := utils.GetCurrentCSV(manifestDir)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(csvFile)
	if err != nil {
		panic(err)
	}

	s := "fieldPath: metadata.annotations\\['olm.targetNamespaces'\\]"
	found, err := regexp.Match(s, b)
	if err != nil {
		return err
	}
	if !found {
		t.Errorf("expected to find %s in CSV", s)
	}

	return nil
}

func TestDoProcessManifest(t *testing.T) {
	type args struct {
		ctx         context.Context
		manifestDir string
	}
	tests := []struct {
		name                 string
		tstCreateManifestDir string
		args                 args
		wantErr              bool
		verify               func(t *testing.T, manifestDir string) error
	}{
		{
			name:                 "olm.targetNamespaces found in CSV (3scale)",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale",
			args: args{
				ctx: context.TODO(),
			},
			verify:  verifyOLMTargetNamespace,
			wantErr: false,
		},
		{
			name:                 "olm.targetNamespaces found in CSV (fuse-online)",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/fuse-online",
			args: args{
				ctx: context.TODO(),
			},
			verify:  verifyOLMTargetNamespace,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.tstCreateManifestDir != "" {
				tstDir := createTempTesManifesttDir(t, tt.tstCreateManifestDir)
				defer os.RemoveAll(tstDir)
				tt.args.manifestDir = tstDir
			}

			if err := DoProcessManifest(tt.args.ctx, tt.args.manifestDir); (err != nil) != tt.wantErr {
				t.Errorf("DoProcessManifest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.verify != nil {
				if err := tt.verify(t, tt.args.manifestDir); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}

		})
	}

}

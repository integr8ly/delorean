package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/integr8ly/delorean/pkg/utils"
)

func mockExtractImageManifests(image string, destDir string) error {
	return nil
}

func createTempTestDir(t *testing.T) string {
	testDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		t.Fatal(err)
	}
	return testDir
}

func createTempTesManifesttDir(t *testing.T, srcManifestDir string) string {
	testDir := createTempTestDir(t)
	err := utils.CopyDirectory(srcManifestDir, testDir)
	if err != nil {
		t.Fatalf("failed to copy the directory %s to %s: %s", srcManifestDir, testDir, err)
	}
	return testDir
}

func TestDoExtractManifests(t *testing.T) {

	type args struct {
		ctx              context.Context
		extractManifests ExtractManifestsCmd
		cmdOpts          *extractManifestsCmdOptions
	}
	tests := []struct {
		name                string
		tstCreateSrcDir     string
		tstCreateDestDir    string
		tstCreateExtractDir bool
		args                args
		wantErr             bool
		verify              func(manifestDir string) error
	}{
		{
			name:             "valid source and destination dir",
			args:             args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{}},
			tstCreateSrcDir:  "../pkg/utils/testdata/validManifests/3scale2",
			tstCreateDestDir: "../pkg/utils/testdata/validManifests/3scale",
			wantErr:          false,
		},
		{
			name: "invalid source dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcDir: "/notreal",
			}},
			tstCreateDestDir: "../pkg/utils/testdata/validManifests/3scale",
			wantErr:          true,
		},
		{
			name: "invalid destination dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				destDir: "/notreal",
			}},
			tstCreateSrcDir: "../pkg/utils/testdata/validManifests/3scale2",
			wantErr:         true,
		},
		{
			name: "invalid args missing source",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcImage: "",
				srcDir:   "",
			}},
			wantErr: true,
		},
		{
			name: "source image",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcImage: "testimage",
			}},
			tstCreateExtractDir: true,
			wantErr:             false,
		},
		{
			name:                "csv filename",
			args:                args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{}},
			tstCreateSrcDir:     "../pkg/utils/testdata/validManifests/crw",
			tstCreateDestDir:    "../pkg/utils/testdata/validManifests/crw",
			tstCreateExtractDir: true,
			wantErr:             false,
			verify: func(destDir string) error {
				expectedCSVFileName := filepath.Join(destDir, "2.1.1", "crwoperator.v2.1.1.clusterserviceversion.yaml")
				_, err := os.Stat(expectedCSVFileName)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.tstCreateExtractDir {
				extractDir := createTempTestDir(t)
				defer os.RemoveAll(extractDir)
				tt.args.cmdOpts.extractDir = extractDir
			}

			if tt.tstCreateDestDir != "" {
				destDir := createTempTesManifesttDir(t, tt.tstCreateDestDir)
				defer os.RemoveAll(destDir)
				tt.args.cmdOpts.destDir = destDir
			}

			if tt.tstCreateSrcDir != "" {
				srcDir := createTempTesManifesttDir(t, tt.tstCreateSrcDir)
				defer os.RemoveAll(srcDir)
				tt.args.cmdOpts.srcDir = srcDir
			}

			if err := DoExtractManifests(tt.args.ctx, tt.args.extractManifests, tt.args.cmdOpts); (err != nil) != tt.wantErr {
				t.Errorf("DoExtractManifests() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.verify != nil {
				if err := tt.verify(tt.args.cmdOpts.destDir); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}

		})
	}
}

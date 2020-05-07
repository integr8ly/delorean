package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/integr8ly/delorean/pkg/utils"
)

func mockExtractImageManifests(image string, destDir string) error {
	return nil
}

func TestDoExtractManifests(t *testing.T) {

	tmpDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	validSrcDir1 := "../pkg/utils/testdata/validManifests/3scale"
	validSrcDir2 := "../pkg/utils/testdata/validManifests/3scale2"
	validDestDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(validDestDir)

	err = utils.CopyDirectory(validSrcDir1, validDestDir)
	if err != nil {
		t.Fatalf("failed to copy the directory %s to %s: %s", validSrcDir1, validDestDir, err)
	}

	type args struct {
		ctx              context.Context
		extractManifests ExtractManifestsCmd
		cmdOpts          *extractManifestsCmdOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid source and destination dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcDir:  validSrcDir2,
				destDir: validDestDir,
			}},
			wantErr: false,
		},
		{
			name: "invalid source dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcDir:  "/notreal",
				destDir: validDestDir,
			}},
			wantErr: true,
		},
		{
			name: "invalid destination dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcDir:  validSrcDir2,
				destDir: "/notreal",
			}},
			wantErr: true,
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
				srcImage:   "testimage",
				extractDir: tmpDir,
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DoExtractManifests(tt.args.ctx, tt.args.extractManifests, tt.args.cmdOpts); (err != nil) != tt.wantErr {
				t.Errorf("DoExtractManifests() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

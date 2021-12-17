package cmd

import (
	"context"
	"os"
	"testing"
)

func TestDoExtractBundle(t *testing.T) {
	var (
		invalidDir       = "./foo/bar"
		validExtractDir  = "./testdata/extractBundleTest/3scale-bundle-extracted"
		validSourceImage = "registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-3scale-rhel7-operator-metadata@sha256:6c916f91899e1c280859ccc49aea9c16554d92e33857151b1050bffc905cb872"
	)
	type args struct {
		ctx                  context.Context
		extractManifests     ExtractManifestsCmd
		extractBundleCmdOpts *extractManifestsCmdOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid source image and valid extract dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcImage:   validSourceImage,
				extractDir: validExtractDir,
			}},
			wantErr: false,
		},
		{
			name: "valid source image and empty extract dir",
			args: args{context.TODO(), mockExtractImageManifests, &extractManifestsCmdOptions{
				srcImage:   validSourceImage,
				extractDir: "",
			}},
			wantErr: false,
		},
		{
			name: "valid source image and invalid extract dir",
			args: args{context.TODO(), ocExtractImage("/"), &extractManifestsCmdOptions{
				srcImage:   validSourceImage,
				extractDir: invalidDir,
			}},
			wantErr: true,
		},
		{
			name: "missing source image and valid extract dir",
			args: args{context.TODO(), ocExtractImage("/"), &extractManifestsCmdOptions{
				srcImage:   "",
				extractDir: validExtractDir,
			}},
			wantErr: true,
		},
		{
			name: "missing source image and invalid extract dir",
			args: args{context.TODO(), ocExtractImage("/"), &extractManifestsCmdOptions{
				srcImage:   "",
				extractDir: invalidDir,
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.extractBundleCmdOpts.extractDir == "" {
				extractDir := createTempTestDir(t)
				defer os.RemoveAll(extractDir)
				tt.args.extractBundleCmdOpts.extractDir = extractDir
			}

			if err := DoExtractManifests(tt.args.ctx, tt.args.extractManifests, tt.args.extractBundleCmdOpts); (err != nil) != tt.wantErr {
				t.Errorf("ExtractBundle error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

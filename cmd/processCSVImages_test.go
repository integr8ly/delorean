package cmd

import (
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func mockProcessCSVImages(manifestDir string, isGa bool, extraImages string) error {
	return nil
}

const (
	testCSV = "../pkg/utils/testdata/validManifests/3scale"
)

func TestProcessCSVImages(t *testing.T) {
	// create a temp manifest directory
	tmpDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = utils.CopyDirectory(testCSV, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx        context.Context
		processCSV ProcessCSVImagesCmd
		cmdOpts    *processCSVImagesCmdOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		verify  func(t *testing.T, err error)
	}{
		{
			name: "Success",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					manifestDir: tmpDir,
					isGa:        false,
					extraImages: []string{},
				}}, wantErr: false,
		},
		{
			name: "Error on missing manifestDir",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					manifestDir: "",
					isGa:        false,
				}}, wantErr: true,
		},
		{
			name: "Ensure image_mirror_file created",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					manifestDir: tmpDir,
					isGa:        false,
					extraImages: nil,
				}},
			verify: func(t *testing.T, err error) {
				if !utils.FileExists(path.Join(tmpDir, utils.MappingFile)) {
					t.Fatal("Error: ", err)
				}
			}, wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DoProcessCSV(tt.args.ctx, tt.args.cmdOpts); (err != nil) != tt.wantErr {
				t.Errorf("buildImageMirrorMappingFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

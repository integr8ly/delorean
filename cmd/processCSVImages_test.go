package cmd

import (
	"bufio"
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"os"
	"path"
	"testing"
)

func mockProcessCSVImages(manifestDir string, isGa bool, extraImages string) error {
	return nil
}

func TestProcessCSVImages(t *testing.T) {

	type args struct {
		ctx        context.Context
		processCSV ProcessCSVImagesCmd
		cmdOpts    *processCSVImagesCmdOptions
	}
	tests := []struct {
		name                 string
		tstCreateManifestDir string
		args                 args
		wantErr              bool
		verify               func(t *testing.T, manifestDir string) error
	}{
		{
			name:                 "Success",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
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
			name:                 "Ensure image_mirror_file created",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					isGa:        false,
					extraImages: nil,
				}},
			verify: func(t *testing.T, manifestDir string) error {
				mappingFile := path.Join(manifestDir, utils.MappingFile)
				if !utils.FileExists(mappingFile) {
					t.Fatal("Error: mapping file does not exist ", mappingFile)
				}
				return nil
			}, wantErr: false,
		},
		{
			name:                 "Ensure image_mirror_file not created",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					isGa: true,
				}},
			verify: func(t *testing.T, manifestDir string) error {
				mappingFile := path.Join(manifestDir, utils.MappingFile)
				if utils.FileExists(mappingFile) {
					t.Fatal("Error: mapping file exists ", mappingFile)
				}
				return nil
			}, wantErr: false,
		},
		{
			name:                 "Ensure image_mirror_file entries are unique",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale2",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					isGa: false,
				}},
			verify: func(t *testing.T, manifestDir string) error {
				mappingFile := path.Join(manifestDir, utils.MappingFile)

				file, err := os.Open(mappingFile)
				if err != nil {
					return err
				}
				defer file.Close()

				var lines []string
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}

				numMappings := len(lines)
				numExpectedMappings := 9 // 3scale2 manifest contains 9 unique images, but 11 references overall
				if numMappings != numExpectedMappings {
					t.Errorf("expected %v image miror mappings, found %v", numExpectedMappings, numMappings)
				}

				return nil
			}, wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.tstCreateManifestDir != "" {
				tstDir := createTempTesManifesttDir(t, tt.tstCreateManifestDir)
				defer os.RemoveAll(tstDir)
				tt.args.cmdOpts.manifestDir = tstDir
			}

			if err := DoProcessCSV(tt.args.ctx, tt.args.cmdOpts); (err != nil) != tt.wantErr {
				t.Errorf("buildImageMirrorMappingFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.verify != nil {
				if err := tt.verify(t, tt.args.cmdOpts.manifestDir); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}
		})
	}

}

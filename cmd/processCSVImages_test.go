package cmd

import (
	"bufio"
	"context"
	"github.com/integr8ly/delorean/pkg/utils"
	"io/ioutil"
	"os"
	"path"
	"regexp"
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
			name:                 "Success 3scale",
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
			name:                 "Success fuse-online",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/fuse-online/7.7.0",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					isGa:        false,
					extraImages: []string{},
				}}, wantErr: false,
		},
		{
			name:                 "Success apicurito",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/apicurito/7.7.0",
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
			name:                 "Ensure image_mirror_file created (isGa=false)",
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
			name:                 "Ensure image_mirror_file not created (isGa=true)",
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
		{
			name:                 "Ensure images converted to delorean (isGA=false)",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale3",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					isGa: false,
				}},
			verify: func(t *testing.T, manifestDir string) error {
				_, csvFile, err := utils.GetCurrentCSV(manifestDir)
				if err != nil {
					return err
				}

				b, err := ioutil.ReadFile(csvFile)
				if err != nil {
					panic(err)
				}

				expectedImages := []string{
					"quay.io/integreatly/delorean:3scale-amp2-backend-rhel7_latest",
					"quay.io/integreatly/delorean:3scale-amp2-apicast-gateway-rhel8_latest",
					"quay.io/integreatly/delorean:3scale-amp2-system-rhel7_latest",
					"quay.io/integreatly/delorean:3scale-amp2-zync-rhel7_latest",
					"quay.io/integreatly/delorean:3scale-amp2-memcached-rhel7_latest",
					"quay.io/integreatly/delorean:rhscl-redis-32-rhel7_latest",
					"quay.io/integreatly/delorean:rhscl-mysql-57-rhel7_latest",
					"quay.io/integreatly/delorean:rhscl-postgresql-10-rhel7_latest",
				}
				for _, image := range expectedImages {
					imageExists, err := regexp.Match(image, b)
					if err != nil {
						return err
					}
					if !imageExists {
						t.Errorf("expected %v to exist in CSV", image)
					}
				}
				return nil
			}, wantErr: false,
		},
		{
			name:                 "Ensure images converted to production (isGA=true)",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale3",
			args: args{
				ctx:        context.TODO(),
				processCSV: mockProcessCSVImages,
				cmdOpts: &processCSVImagesCmdOptions{
					isGa: true,
				}},
			verify: func(t *testing.T, manifestDir string) error {
				_, csvFile, err := utils.GetCurrentCSV(manifestDir)
				if err != nil {
					return err
				}

				b, err := ioutil.ReadFile(csvFile)
				if err != nil {
					panic(err)
				}

				expectedImages := []string{
					"registry.redhat.io/3scale-amp2/backend-rhel7@sha256:d8322db4149afc5672ebc3d0430a077c58a8e3e98d7fce720b6a5a3d2498c9c5",
					"registry.redhat.io/3scale-amp2/apicast-gateway-rhel8@sha256:52013cc8722ce507e3d0b066a8ae4edb930fb54e24e9f653016658ad1708b5d7",
					"registry.redhat.io/3scale-amp2/system-rhel7@sha256:a934997501b41be2ca2b62e37c35bd334252b5e2ed28652c275bd1de8a9d324a",
					"registry.redhat.io/3scale-amp2/zync-rhel7@sha256:34fa60de75f5a0e220105c6bf0ed676f16c8b206812fad65078cf98a16a6d4ef",
					"registry.redhat.io/3scale-amp2/memcached-rhel7@sha256:2be57d773843135c0677e31d34b0cd24fa9dafc4ef1367521caa2bab7c6122e6",
					"registry.redhat.io/rhscl/redis-32-rhel7@sha256:a9bdf52384a222635efc0284db47d12fbde8c3d0fcb66517ba8eefad1d4e9dc9",
					"registry.redhat.io/rhscl/mysql-57-rhel7@sha256:9a781abe7581cc141e14a7e404ec34125b3e89c008b14f4e7b41e094fd3049fe",
					"registry.redhat.io/rhscl/postgresql-10-rhel7@sha256:de3ab628b403dc5eed986a7f392c34687bddafee7bdfccfd65cecf137ade3dfd",
					"registry.redhat.io/3scale-amp2/3scale-rhel7-operator@sha256:1ba6ec8ed984a011796bbe1eafabb2791957f58ed66ec4a484c024dd96eaf427",
				}
				for _, image := range expectedImages {
					imageExists, err := regexp.Match(image, b)
					if err != nil {
						return err
					}
					if !imageExists {
						t.Errorf("expected %v to exist in CSV", image)
					}
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

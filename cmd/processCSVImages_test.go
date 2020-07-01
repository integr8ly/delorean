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
			name:                 "Ensure image_mirror_file entries are unique and sorted",
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

				expectedMappings := []string{
					"registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-3scale-rhel7-operator@sha256:2ba16314ee046b3c3814fe4e356b728da6853743bd72f8651e1a338e8bbf4f81 quay.io/integreatly/delorean:3scale-amp2-3scale-rhel7-operator_2ba16314ee046b3c3814fe4e356b728da6853743bd72f8651e1a338e8bbf4f81",
					"registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-apicast-gateway-rhel8@sha256:21be62a6557846337dc0cf764be63442718fab03b95c198a301363886a9e74f9 quay.io/integreatly/delorean:3scale-amp2-apicast-gateway-rhel8_21be62a6557846337dc0cf764be63442718fab03b95c198a301363886a9e74f9",
					"registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-backend-rhel7@sha256:ea8a31345d3c2a56b02998b019db2e17f61eeaa26790a07962d5e3b66032d8e5 quay.io/integreatly/delorean:3scale-amp2-backend-rhel7_ea8a31345d3c2a56b02998b019db2e17f61eeaa26790a07962d5e3b66032d8e5",
					"registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-memcached-rhel7@sha256:ff5f3d2d131631d5db8985a5855ff4607e91f0aa86d07dafdcec4f7da13c9e05 quay.io/integreatly/delorean:3scale-amp2-memcached-rhel7_ff5f3d2d131631d5db8985a5855ff4607e91f0aa86d07dafdcec4f7da13c9e05",
					"registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-system-rhel7@sha256:93819c324831353bb8f7cb6e9910694b88609c3a20d4c1b9a22d9c2bbfbad16f quay.io/integreatly/delorean:3scale-amp2-system-rhel7_93819c324831353bb8f7cb6e9910694b88609c3a20d4c1b9a22d9c2bbfbad16f",
					"registry-proxy.engineering.redhat.com/rh-osbs/3scale-amp2-zync-rhel7@sha256:f4d5c1fdebe306f4e891ddfc4d3045a622d2f01db21ecfc9397cab25c9baa91a quay.io/integreatly/delorean:3scale-amp2-zync-rhel7_f4d5c1fdebe306f4e891ddfc4d3045a622d2f01db21ecfc9397cab25c9baa91a",
					"registry-proxy.engineering.redhat.com/rh-osbs/rhscl-mysql-57-rhel7@sha256:9a781abe7581cc141e14a7e404ec34125b3e89c008b14f4e7b41e094fd3049fe quay.io/integreatly/delorean:rhscl-mysql-57-rhel7_9a781abe7581cc141e14a7e404ec34125b3e89c008b14f4e7b41e094fd3049fe",
					"registry-proxy.engineering.redhat.com/rh-osbs/rhscl-postgresql-10-rhel7@sha256:de3ab628b403dc5eed986a7f392c34687bddafee7bdfccfd65cecf137ade3dfd quay.io/integreatly/delorean:rhscl-postgresql-10-rhel7_de3ab628b403dc5eed986a7f392c34687bddafee7bdfccfd65cecf137ade3dfd",
					"registry-proxy.engineering.redhat.com/rh-osbs/rhscl-redis-32-rhel7@sha256:a9bdf52384a222635efc0284db47d12fbde8c3d0fcb66517ba8eefad1d4e9dc9 quay.io/integreatly/delorean:rhscl-redis-32-rhel7_a9bdf52384a222635efc0284db47d12fbde8c3d0fcb66517ba8eefad1d4e9dc9",
				}

				numMappings := len(lines)
				numExpectedMappings := len(expectedMappings) // 3scale2 manifest contains 9 unique images, but 11 references overall
				if numMappings != len(expectedMappings) {
					t.Errorf("expected %v image miror mappings, found %v", numExpectedMappings, numMappings)
				}

				for i, line := range lines {
					if expectedMappings[i] != line {
						t.Errorf("expected %s at line %v, found %s", expectedMappings[i], i, line)
					}
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
					"quay.io/integreatly/delorean:3scale-amp2-backend-rhel7_d8322db4149afc5672ebc3d0430a077c58a8e3e98d7fce720b6a5a3d2498c9c5",
					"quay.io/integreatly/delorean:3scale-amp2-apicast-gateway-rhel8_52013cc8722ce507e3d0b066a8ae4edb930fb54e24e9f653016658ad1708b5d7",
					"quay.io/integreatly/delorean:3scale-amp2-system-rhel7_a934997501b41be2ca2b62e37c35bd334252b5e2ed28652c275bd1de8a9d324a",
					"quay.io/integreatly/delorean:3scale-amp2-zync-rhel7_34fa60de75f5a0e220105c6bf0ed676f16c8b206812fad65078cf98a16a6d4ef",
					"quay.io/integreatly/delorean:3scale-amp2-memcached-rhel7_2be57d773843135c0677e31d34b0cd24fa9dafc4ef1367521caa2bab7c6122e6",
					"quay.io/integreatly/delorean:rhscl-redis-32-rhel7_a9bdf52384a222635efc0284db47d12fbde8c3d0fcb66517ba8eefad1d4e9dc9",
					"quay.io/integreatly/delorean:rhscl-mysql-57-rhel7_9a781abe7581cc141e14a7e404ec34125b3e89c008b14f4e7b41e094fd3049fe",
					"quay.io/integreatly/delorean:rhscl-postgresql-10-rhel7_de3ab628b403dc5eed986a7f392c34687bddafee7bdfccfd65cecf137ade3dfd",
					"quay.io/integreatly/delorean:3scale-amp2-3scale-rhel7-operator_1ba6ec8ed984a011796bbe1eafabb2791957f58ed66ec4a484c024dd96eaf427",
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
		{
			name:                 "Ensure relatedImages converted to production (isGA=true)",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/crw",
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

				relatedImages := []string{
					"registry.stage.redhat.io/codeready-workspaces/crw-2-rhel8-operator@sha256:02e8777fa295e6615bbd73f3d92911e7e7029b02cdf6346eba502aaeb8fe3de1",
					"registry.stage.redhat.io/codeready-workspaces/server-rhel8@sha256:f7b27fb525a24c4273f0a3e18461a70f3cbb897e845e44abd8ca10fd1de3e1b2",
					"registry.stage.redhat.io/codeready-workspaces/pluginregistry-rhel8@sha256:6cd737a9e9df54407959a0e8e4bb6a3e3b9e37cf590193545f609b2c4af4bf46",
					"registry.stage.redhat.io/codeready-workspaces/devfileregistry-rhel8@sha256:0124562131e8cde6b2b9a5e4bced93522da3c1c95e9122306ecd8acb093650e0",
					"registry.stage.redhat.io/ubi8-minimal@sha256:9285da611437622492f9ef4229877efe302589f1401bbd4052e9bb261b3d4387",
					"registry.stage.redhat.io/rhscl/postgresql-96-rhel7@sha256:196abd9a1221fb38dd5693203f068fc4d520bb351928ef84e5e15984f5152476",
					"registry.stage.redhat.io/redhat-sso-7/sso73-openshift@sha256:0dc950903bbc971c14e6223efe3493f0f50eb8af7cbe91aeea621f80f99f155f",
					"registry.stage.redhat.io/codeready-workspaces/pluginbroker-metadata-rhel8@sha256:6c9abe63a70a6146dc49845f2f7732e3e6e0bcae6a19c3a6557367d6965bc1f8",
					"registry.stage.redhat.io/codeready-workspaces/pluginbroker-artifacts-rhel8@sha256:5815bab69fc343cbf6dac0fd67dd70a25757fac08689a15e4a762655fa2e8a2c",
				}
				for _, image := range relatedImages {
					imageFound, err := regexp.Match(image, b)
					if err != nil {
						return err
					}
					if imageFound {
						t.Errorf("found %v in relatedImages", image)
					}
				}
				return nil
			}, wantErr: false,
		},
		{
			name:                 "Ensure relatedImages set correctly when mixed (isGA=true)",
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

				relatedStagingImages := []string{
					"registry.stage.redhat.io/3scale-amp2/apicast-gateway-rhel8@sha256:52013cc8722ce507e3d0b066a8ae4edb930fb54e24e9f653016658ad1708b5d7",
					"registry.stage.redhat.io/3scale-amp2/backend-rhel7@sha256:d8322db4149afc5672ebc3d0430a077c58a8e3e98d7fce720b6a5a3d2498c9c5",
					"registry.stage.redhat.io/3scale-amp2/system-rhel7@sha256:a934997501b41be2ca2b62e37c35bd334252b5e2ed28652c275bd1de8a9d324a",
					"registry.stage.redhat.io/3scale-amp2/zync-rhel7@sha256:34fa60de75f5a0e220105c6bf0ed676f16c8b206812fad65078cf98a16a6d4ef",
					"registry.stage.redhat.io/3scale-amp2/memcached-rhel7@sha256:2be57d773843135c0677e31d34b0cd24fa9dafc4ef1367521caa2bab7c6122e6",
				}
				relatedProdImages := []string{
					"registry.redhat.io/rhscl/redis-32-rhel7@sha256:a9bdf52384a222635efc0284db47d12fbde8c3d0fcb66517ba8eefad1d4e9dc9",
					"registry.redhat.io/rhscl/mysql-57-rhel7@sha256:9a781abe7581cc141e14a7e404ec34125b3e89c008b14f4e7b41e094fd3049fe",
					"registry.redhat.io/rhscl/postgresql-10-rhel7@sha256:de3ab628b403dc5eed986a7f392c34687bddafee7bdfccfd65cecf137ade3dfd",
				}

				for _, image := range relatedStagingImages {
					imageFound, err := regexp.Match(image, b)
					if err != nil {
						return err
					}
					if imageFound {
						t.Errorf("found %v in relatedImages", image)
					}
				}

				for _, image := range relatedProdImages {
					imageFound, err := regexp.Match(image, b)
					if err != nil {
						return err
					}
					if !imageFound {
						t.Errorf("expected to find %v in relatedImages", image)
					}
				}
				return nil
			}, wantErr: false,
		},
		{
			name:                 "Ensure relatedImages field not added",
			tstCreateManifestDir: "../pkg/utils/testdata/validManifests/3scale",
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

				fieldFound, err := regexp.Match("relatedImages", b)
				if err != nil {
					return err
				}
				if fieldFound {
					t.Errorf("found relatedImages field in CSV")
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

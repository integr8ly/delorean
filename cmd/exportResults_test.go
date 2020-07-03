package cmd

import (
	"context"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/utils"
	"io/ioutil"
	"os"
	"testing"
)

func TestExportResultsCmd(t *testing.T) {
	zipFile := "test-xyz.zip"
	tmpDir, err := ioutil.TempDir(os.TempDir(), "export-*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tmpDir = tmpDir + "/"
	defer os.RemoveAll(tmpDir) // clean up

	cases := []struct {
		description string
		zipFile     string
		tmpDir      string
		uploader    s3manageriface.UploaderAPI
	}{
		{
			description: "should generate metadata.json, zip it and export it",
			zipFile:     zipFile,
			tmpDir:      tmpDir,
			uploader: &utils.MockS3Uploader{
				UploadFunc: func(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
					output := s3manager.UploadOutput{
						Location: *input.Key,
					}
					return &output, nil
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {

			cmd := &exportCmd{
				metadataFile: c.tmpDir + metadataFileName,
				metadataObj: MetaData{
					Name: "test", Version: "x.y.z", Job: "xy",
				},
				zippedDir: c.tmpDir,
				zipFile:   c.zipFile,
				bucket:    "test-bucket",
				uploader:  c.uploader,
			}
			err = cmd.run(context.TODO())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err = os.Stat(c.tmpDir + metadataFileName); err != nil {
				t.Fatalf("generated %v not found", metadataFileName)
			}
			if _, err = os.Stat(c.tmpDir + c.zipFile); err != nil {
				t.Fatalf("generated %v not found", c.zipFile)
			}
		})
	}
}

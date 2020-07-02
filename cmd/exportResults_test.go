package cmd

import (
	"context"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/utils"
	"os"
	"testing"
)

func TestExportResultsCmd(t *testing.T) {
	cases := []struct {
		description string
		uploader    s3manageriface.UploaderAPI
	}{
		{
			description: "should generate metadata.json, zip it and export it",
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
			tmpDir := "/tmp/export-unit-test/"
			zipFile := "test-xyz.zip"
			os.MkdirAll(tmpDir, os.ModePerm)
			cmd := &exportCmd{
				metadataFile: tmpDir + metadataFileName,
				metadataObj: MetaData{
					Name: "test", Version: "x.y.z", Job: "xy",
				},
				zippedDir: tmpDir,
				zipFile:   zipFile,
				bucket:    "test-bucket",
				uploader:  c.uploader,
			}
			err := cmd.run(context.TODO())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err = os.Stat(tmpDir + metadataFileName); err != nil {
				t.Fatalf("generated %v not found", metadataFileName)
			}
			if _, err = os.Stat(tmpDir + zipFile); err != nil {
				t.Fatalf("generated %v not found", zipFile)
			}
		})
	}
}

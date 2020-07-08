package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/polarion"
	"github.com/integr8ly/delorean/pkg/utils"
)

type mockPolarionXUnitImporter struct {
	importf      func(xunit *polarion.PolarionXUnit) (int, error)
	getJobStatus func(id int) (polarion.XUnitJobStatus, error)
}

func (i *mockPolarionXUnitImporter) Import(xunit *polarion.PolarionXUnit) (int, error) {
	return i.importf(xunit)
}

func (i *mockPolarionXUnitImporter) GetJobStatus(id int) (polarion.XUnitJobStatus, error) {
	return i.getJobStatus(id)
}

func TestPolarionImportCmd(t *testing.T) {
	cases := []struct {
		description      string
		s3               s3iface.S3API
		s3downloader     s3manageriface.DownloaderAPI
		polarionImporter func() polarion.XUnitImporterService
		expectError      bool
	}{
		{
			description: "success",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj1 := &s3.Object{
						Key: aws.String("results.zip"),
					}
					obj2 := &s3.Object{
						Key: aws.String("results"),
					}
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{obj1, obj2},
					}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					t := []*s3.Tag{}
					return &s3.GetObjectTaggingOutput{
						TagSet: t,
					}, nil
				},
				PutObjTaggingFunc: func(input *s3.PutObjectTaggingInput) (output *s3.PutObjectTaggingOutput, err error) {
					if !hasTag(input.Tagging.TagSet, polarionTagKey, polarionTagVal) {
						return nil, fmt.Errorf("missing expected tag: %s=%s", polarionTagKey, polarionTagVal)
					}
					return &s3.PutObjectTaggingOutput{}, nil
				},
			},
			s3downloader: &utils.MockS3Downloader{
				DownloadFunc: func(o io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					content, err := ioutil.ReadFile("./testdata/rhmi-install-addon-flow.zip")
					if err != nil {
						return 0, err
					}
					b, err := o.WriteAt(content, 0)
					return int64(b), err
				},
			},
			polarionImporter: func() polarion.XUnitImporterService {

				status := polarion.XUnitJobStatus("")

				return &mockPolarionXUnitImporter{
					importf: func(xunit *polarion.PolarionXUnit) (int, error) {

						var got string
						for _, p := range xunit.Properties {
							if p.Name == "polarion-testrun-title" {
								got = p.Value
							}
						}
						if expected := "RHMI 2.4.0-rc1 rhmi-install-addon-flow Automated Tests"; got != expected {
							return 0, fmt.Errorf("the test run title '%s' is not equal to the expected title '%s'", got, expected)
						}

						return 1, nil
					},
					getJobStatus: func(id int) (polarion.XUnitJobStatus, error) {

						if status == "" {
							status = polarion.ReadyStatus

							go func() {
								// wait 3s before reporting Running
								time.Sleep(3 * time.Second)
								status = polarion.RunningStatus

								// wait other 3s before reporting Success
								time.Sleep(3 * time.Second)
								status = polarion.SuccessStatus
							}()
						}

						// the status will automatically update to Sucess after 6s
						return status, nil
					},
				}
			},
			expectError: false,
		},
		{
			description: "skip import",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj := &s3.Object{Key: aws.String("results.zip")}
					return &s3.ListObjectsV2Output{Contents: []*s3.Object{obj}}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					return &s3.GetObjectTaggingOutput{TagSet: []*s3.Tag{}}, nil
				},
				PutObjTaggingFunc: func(input *s3.PutObjectTaggingInput) (output *s3.PutObjectTaggingOutput, err error) {
					return &s3.PutObjectTaggingOutput{}, nil
				},
			},
			s3downloader: &utils.MockS3Downloader{
				DownloadFunc: func(o io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					content, err := ioutil.ReadFile("./testdata/nightly-rhmi-install-addon-flow.zip")
					if err != nil {
						return 0, err
					}
					b, err := o.WriteAt(content, 0)
					return int64(b), err
				},
			},
			polarionImporter: func() polarion.XUnitImporterService {
				return &mockPolarionXUnitImporter{
					importf: func(xunit *polarion.PolarionXUnit) (int, error) {
						return 0, errors.New("unexpected call to import()")
					},
					getJobStatus: func(id int) (polarion.XUnitJobStatus, error) {
						return polarion.SuccessStatus, errors.New("unexpected call to getJobStatus()")
					},
				}
			},
			expectError: false,
		},
		{
			description: "simulate error",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj := &s3.Object{Key: aws.String("results.zip")}
					return &s3.ListObjectsV2Output{Contents: []*s3.Object{obj}}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					return &s3.GetObjectTaggingOutput{TagSet: []*s3.Tag{}}, nil
				},
				PutObjTaggingFunc: func(input *s3.PutObjectTaggingInput) (output *s3.PutObjectTaggingOutput, err error) {
					return &s3.PutObjectTaggingOutput{}, nil
				},
			},
			s3downloader: &utils.MockS3Downloader{
				DownloadFunc: func(o io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					content, err := ioutil.ReadFile("./testdata/rhmi-install-addon-flow.zip")
					if err != nil {
						return 0, err
					}
					b, err := o.WriteAt(content, 0)
					return int64(b), err
				},
			},
			polarionImporter: func() polarion.XUnitImporterService {
				return &mockPolarionXUnitImporter{
					importf: func(xunit *polarion.PolarionXUnit) (int, error) {
						return 0, errors.New("simulated error")
					},
				}
			},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &polarionImportCmd{
				s3:               c.s3,
				s3downloader:     c.s3downloader,
				fromBucket:       "test-bucket",
				polarionImporter: c.polarionImporter(),
			}
			err := cmd.run(context.TODO())

			if err != nil && !c.expectError {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.expectError {
				t.Fatalf("expect to have error but it's nil")
			}
		})
	}
}

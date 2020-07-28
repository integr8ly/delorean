package cmd

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatahubImportCmd(t *testing.T) {
	pgwOK := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer pgwOK.Close()

	var cases = []struct {
		description  string
		s3           s3iface.S3API
		s3downloader s3manageriface.DownloaderAPI
		expectError  bool
		gateway      string
		reportName   string
	}{
		{
			description: "push success",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj := &s3.Object{
						Key: aws.String("downtime-report.yaml"),
					}
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{obj},
					}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					t := []*s3.Tag{}
					return &s3.GetObjectTaggingOutput{
						TagSet: t,
					}, nil
				},
				PutObjTaggingFunc: func(input *s3.PutObjectTaggingInput) (output *s3.PutObjectTaggingOutput, err error) {
					if !hasTag(input.Tagging.TagSet, datahubTagKey, datahubTagVal) {
						return nil, fmt.Errorf("missing expected tag: %s=%s", datahubTagKey, datahubTagVal)
					}
					return &s3.PutObjectTaggingOutput{}, nil
				},
			},
			s3downloader: &utils.MockS3Downloader{
				DownloadFunc: func(o io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					content, err := ioutil.ReadFile("./testdata/queryReport/downtime-report.yaml")
					if err != nil {
						return 0, err
					}
					b, err := o.WriteAt(content, 0)
					return int64(b), err
				},
			},
			expectError: false,
			gateway:     pgwOK.URL,
			reportName:  "downtime-report.yaml",
		},
		{
			description: "push gateway failure",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj := &s3.Object{
						Key: aws.String("downtime-report.yaml"),
					}
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{obj},
					}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					t := []*s3.Tag{}
					return &s3.GetObjectTaggingOutput{
						TagSet: t,
					}, nil
				},
				PutObjTaggingFunc: func(input *s3.PutObjectTaggingInput) (output *s3.PutObjectTaggingOutput, err error) {
					if !hasTag(input.Tagging.TagSet, datahubTagKey, datahubTagVal) {
						return nil, fmt.Errorf("missing expected tag: %s=%s", datahubTagKey, datahubTagVal)
					}
					return &s3.PutObjectTaggingOutput{}, nil
				},
			},
			s3downloader: &utils.MockS3Downloader{
				DownloadFunc: func(o io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					content, err := ioutil.ReadFile("./testdata/queryReport/downtime-report.yaml")
					if err != nil {
						return 0, err
					}
					b, err := o.WriteAt(content, 0)
					return int64(b), err
				},
			},
			expectError: true,
			gateway:     "not-a-real-gateway",
			reportName:  "downtime-report.yaml",
		},
		{
			description: "invalid file failure",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj := &s3.Object{
						Key: aws.String("downtime-report.yaml"),
					}
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{obj},
					}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					t := []*s3.Tag{}
					return &s3.GetObjectTaggingOutput{
						TagSet: t,
					}, nil
				},
			},
			s3downloader: &utils.MockS3Downloader{
				DownloadFunc: func(o io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					content, err := ioutil.ReadFile("./testdata/queryReport/downtime-report2.yaml")
					if err != nil {
						return 0, err
					}
					b, err := o.WriteAt(content, 0)
					return int64(b), err
				},
			},
			expectError: true,
			gateway:     pgwOK.URL,
			reportName:  "not-a-report.yaml",
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &datahubImportCmd{
				s3:           c.s3,
				s3Downloader: c.s3downloader,
				fromBucket:   "test-bucket",
				reportName:   c.reportName,
				pushgateway:  c.gateway,
				jobName:      "testJob",
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

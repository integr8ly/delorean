package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/reportportal"
	"github.com/integr8ly/delorean/pkg/utils"
	"io"
	"io/ioutil"
	"testing"
)

type mockRPLaunchService struct {
	reportportal.RPLaunchManager
	importFuncInvoked bool
	importFunc        func(string, string) (*reportportal.RPLaunchResponse, error)
	updateFuncInvoked bool
	updateFunc        func(string, string, *reportportal.RPLaunchUpdateInput) (*reportportal.RPLaunchResponse, error)
}

func (m *mockRPLaunchService) Import(_ context.Context, projectName string, importFile string, _ string) (*reportportal.RPLaunchResponse, error) {
	m.importFuncInvoked = true
	return m.importFunc(projectName, importFile)
}

func (m *mockRPLaunchService) Update(_ context.Context, projectName string, launchId string, input *reportportal.RPLaunchUpdateInput) (*reportportal.RPLaunchResponse, error) {
	m.updateFuncInvoked = true
	return m.updateFunc(projectName, launchId, input)
}

func (m *mockRPLaunchService) Validate() error {
	if !m.importFuncInvoked {
		return errors.New("Import func is not invoked")
	}
	if !m.updateFuncInvoked {
		return errors.New("Update func is not invoked")
	}
	return nil
}

func TestReportPortalImportCmd(t *testing.T) {
	testLaunchId := "testLaunchId"
	testLaunchIdMsg := fmt.Sprintf("launch id = %s is updated", testLaunchId)
	cases := []struct {
		description  string
		s3           s3iface.S3API
		s3downloader s3manageriface.DownloaderAPI
		rpLaunch     *mockRPLaunchService
		expectError  bool
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
					if !hasTag(input.Tagging.TagSet, reportPortalTagKey, reportPortalTagVal) {
						return nil, fmt.Errorf("missing expected tag: %s=%s", reportPortalTagKey, reportPortalTagVal)
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
			rpLaunch: &mockRPLaunchService{
				importFunc: func(s string, s2 string) (response *reportportal.RPLaunchResponse, err error) {
					return &reportportal.RPLaunchResponse{Msg: testLaunchIdMsg}, nil
				},
				updateFunc: func(project string, launchId string, input *reportportal.RPLaunchUpdateInput) (response *reportportal.RPLaunchResponse, err error) {
					if launchId != testLaunchId {
						return nil, fmt.Errorf("wrong launch id. expected %s, but got: %s", testLaunchId, launchId)
					}
					return &reportportal.RPLaunchResponse{Msg: testLaunchId}, nil
				},
			},
			expectError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &reportPortalImportCmd{
				s3:              c.s3,
				s3downloader:    c.s3downloader,
				rpLaunchService: c.rpLaunch,
				fromBucket:      "test-bucket",
				rpProjectName:   "test-project",
			}
			err := cmd.run(context.TODO())

			rpErr := c.rpLaunch.Validate()
			if rpErr != nil {
				t.Fatalf("unexpected error: %v", rpErr)
			}

			if err != nil && !c.expectError {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.expectError {
				t.Fatalf("expect to have error but it's nil")
			}
		})
	}
}

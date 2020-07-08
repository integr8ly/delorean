package cmd

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/utils"
	"testing"
)

func TestCleanupReportsCmd(t *testing.T) {
	copiedFiles := 0
	deleteFiles := 0
	cases := []struct {
		description        string
		s3                 s3iface.S3API
		s3Deleter          s3manageriface.BatchDelete
		config             cleanupConfigList
		expectCopifedFiles int
		expectDeletedFiles int
		expectError        bool
	}{
		{
			description: "success",
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					obj1 := &s3.Object{
						Key: aws.String("obj1.zip"),
					}
					obj2 := &s3.Object{
						Key: aws.String("obj2.zip"),
					}
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{obj1, obj2},
					}, nil
				},
				GetObjTaggingFunc: func(input *s3.GetObjectTaggingInput) (output *s3.GetObjectTaggingOutput, err error) {
					if *input.Key == "obj1.zip" {
						tags := []*s3.Tag{
							{
								Key:   aws.String("tag1"),
								Value: aws.String("true"),
							},
							{
								Key:   aws.String("tag2"),
								Value: aws.String("true"),
							},
						}
						return &s3.GetObjectTaggingOutput{
							TagSet: tags,
						}, nil
					} else {
						tags := []*s3.Tag{
							{
								Key:   aws.String("tag1"),
								Value: aws.String("true"),
							},
							{
								Key:   aws.String("tag2"),
								Value: aws.String("false"),
							},
						}
						return &s3.GetObjectTaggingOutput{
							TagSet: tags,
						}, nil
					}
				},
				CopyObjectFunc: func(input *s3.CopyObjectInput) (output *s3.CopyObjectOutput, err error) {
					copiedFiles++
					return &s3.CopyObjectOutput{}, nil
				},
			},
			s3Deleter: &utils.MockS3BatchDeleter{
				BatchDeleteFunc: func(iterator s3manager.BatchDeleteIterator) error {
					for ok := iterator.Next(); ok; ok = iterator.Next() {
						deleteFiles++
					}
					return nil
				},
			},
			expectCopifedFiles: 1,
			expectDeletedFiles: 1,
			expectError:        false,
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			copiedFiles = 0
			deleteFiles = 0
			config := &cleanupConfigList{Configs: []cleanupConfig{
				{
					Bucket: "source",
					Tags: []objectTag{
						{
							Key:   "tag1",
							Value: "true",
						},
						{
							Key:   "tag2",
							Value: "true",
						},
					},
				},
			}}
			cmd := &cleanupReportsCmd{
				config:    config,
				s3:        c.s3,
				s3Deleter: c.s3Deleter,
			}
			err := cmd.run(context.TODO())
			if err != nil && !c.expectError {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.expectError {
				t.Fatal("error is expected but got nil")
			}
			if copiedFiles != c.expectCopifedFiles {
				t.Fatalf("expect %d files copied but got %d", c.expectCopifedFiles, copiedFiles)
			}
			if deleteFiles != c.expectDeletedFiles {
				t.Fatalf("expect %d files deleted but got %d", c.expectDeletedFiles, deleteFiles)
			}
		})
	}
}

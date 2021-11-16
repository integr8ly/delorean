package cmd

import (
	"context"
	"testing"

	"github.com/integr8ly/cluster-service/pkg/clusterservice"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/integr8ly/delorean/pkg/utils"
)

type clusterServiceMock struct {
	clusterservice.Client
}

func (c *clusterServiceMock) DeleteResourcesForCluster(_ string, _ map[string]string, _ bool) (*clusterservice.Report, error) {
	return &clusterservice.Report{Items: []*clusterservice.ReportItem{{ActionStatus: "complete"}}}, nil
}

func TestCleanupAwsCmd(t *testing.T) {

	cases := []struct {
		description string
		dryRun      bool
		ec2         ec2iface.EC2API
		s3          s3iface.S3API

		expectedNumDeletedResources int
		expectError                 bool
	}{
		{
			description:                 "no resources should be deleted",
			dryRun:                      false,
			expectedNumDeletedResources: 0,
			expectError:                 false,
			ec2: &utils.MockEC2API{
				DescribeInstancesOutput: &ec2.DescribeInstancesOutput{
					Reservations: []*ec2.Reservation{
						{
							Instances: []*ec2.Instance{
								{
									InstanceId: aws.String("Running OSD instance"),
									Tags: []*ec2.Tag{
										{Key: aws.String("kubernetes.io/cluster/dont-delete-me"), Value: aws.String("owned")},
									},
								},
							},
						},
					},
				},
				DescribeVpcsOutput: &ec2.DescribeVpcsOutput{
					Vpcs: []*ec2.Vpc{
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("kubernetes.io/cluster/dont-delete-me"), Value: aws.String("owned")},
							},
							VpcId: aws.String("osd-vpc-attached-to-ec2"),
						},
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("integreatly.org/clusterID"), Value: aws.String("dont-delete-me")},
							},
							VpcId: aws.String("rhmi-vpc-attached-to-running-osd-cluster"),
						},
						{
							IsDefault: aws.Bool(true),
							Tags:      nil,
							VpcId:     aws.String("empty-vpc"),
						},
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("some/random"), Value: aws.String("tag-name")},
							},
							VpcId: aws.String("osd-unrelated-nonempty-vpc"),
						},
					},
				},
			},
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{
							{
								Key: aws.String("testobj.zip"),
							},
						},
					}, nil
				},
				ListBucketsOutput: &s3.ListBucketsOutput{
					Buckets: []*s3.Bucket{
						{
							Name: aws.String("managed-velero-backups"),
						},
						{
							Name: aws.String("rhmi-bucket"),
						},
						{
							Name: aws.String("other-bucket"),
						},
					},
				},
				GetBucketTaggingFunc: func(input *s3.GetBucketTaggingInput) (output *s3.GetBucketTaggingOutput, err error) {
					t.Log("GetBucketTaggingFunc: bucket name:", *input.Bucket)

					if aws.StringValue(input.Bucket) == "managed-velero-backups" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("velero.io/infrastructureName"),
									Value: aws.String("dont-delete-me"),
								},
							}}, nil
					} else if aws.StringValue(input.Bucket) == "rhmi-bucket" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("integreatly.org/clusterID"),
									Value: aws.String("dont-delete-me"),
								},
							}}, nil
					} else if aws.StringValue(input.Bucket) == "other-bucket" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("some/random"),
									Value: aws.String("tag"),
								},
							}}, nil
					}
					return nil, nil
				},
			},
		},
		{
			description:                 "all resources should be deleted",
			dryRun:                      false,
			expectedNumDeletedResources: 5,
			expectError:                 false,
			ec2: &utils.MockEC2API{
				DescribeInstancesOutput: &ec2.DescribeInstancesOutput{
					Reservations: []*ec2.Reservation{
						{
							Instances: []*ec2.Instance{},
						},
					},
				},
				DescribeVpcsOutput: &ec2.DescribeVpcsOutput{
					Vpcs: []*ec2.Vpc{
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("kubernetes.io/cluster/delete-me"), Value: aws.String("owned")},
							},
							VpcId: aws.String("osd-vpc-unattached"),
						},
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("integreatly.org/clusterID"), Value: aws.String("delete-me")},
							},
							VpcId: aws.String("rhmi-vpc-unattached"),
						},
						{
							IsDefault: aws.Bool(false),
							Tags:      nil,
							VpcId:     aws.String("empty-vpc"),
						},
					},
				},
			},
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{
							{
								Key: aws.String("testobj.zip"),
							},
						},
					}, nil
				},
				ListBucketsOutput: &s3.ListBucketsOutput{
					Buckets: []*s3.Bucket{
						{
							Name: aws.String("managed-velero-backups"),
						},
						{
							Name: aws.String("rhmi-bucket"),
						},
					},
				},
				GetBucketTaggingFunc: func(input *s3.GetBucketTaggingInput) (output *s3.GetBucketTaggingOutput, err error) {

					if aws.StringValue(input.Bucket) == "managed-velero-backups" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("velero.io/infrastructureName"),
									Value: aws.String("delete-me"),
								},
							}}, nil
					} else if aws.StringValue(input.Bucket) == "rhmi-bucket" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("integreatly.org/clusterID"),
									Value: aws.String("delete-me"),
								},
							}}, nil
					}
					return nil, nil
				},
			},
		},
		{
			description:                 "nothing should get deleted with dry run set to true",
			dryRun:                      true,
			expectedNumDeletedResources: 0,
			expectError:                 false,
			ec2: &utils.MockEC2API{
				DescribeInstancesOutput: &ec2.DescribeInstancesOutput{
					Reservations: []*ec2.Reservation{
						{
							Instances: []*ec2.Instance{},
						},
					},
				},
				DescribeVpcsOutput: &ec2.DescribeVpcsOutput{
					Vpcs: []*ec2.Vpc{
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("kubernetes.io/cluster/delete-me"), Value: aws.String("owned")},
							},
							VpcId: aws.String("osd-vpc-unattached"),
						},
						{
							IsDefault: aws.Bool(false),
							Tags: []*ec2.Tag{
								{Key: aws.String("integreatly.org/clusterID"), Value: aws.String("delete-me")},
							},
							VpcId: aws.String("rhmi-vpc-unattached"),
						},
						{
							IsDefault: aws.Bool(false),
							Tags:      nil,
							VpcId:     aws.String("empty-vpc"),
						},
					},
				},
			},
			s3: &utils.MockS3API{
				ListObjsFunc: func(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
					return &s3.ListObjectsV2Output{
						Contents: []*s3.Object{
							{
								Key: aws.String("testobj.zip"),
							},
						},
					}, nil
				},
				ListBucketsOutput: &s3.ListBucketsOutput{
					Buckets: []*s3.Bucket{
						{
							Name: aws.String("managed-velero-backups"),
						},
						{
							Name: aws.String("rhmi-bucket"),
						},
					},
				},
				GetBucketTaggingFunc: func(input *s3.GetBucketTaggingInput) (output *s3.GetBucketTaggingOutput, err error) {

					if aws.StringValue(input.Bucket) == "managed-velero-backups" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("velero.io/infrastructureName"),
									Value: aws.String("delete-me"),
								},
							}}, nil
					} else if aws.StringValue(input.Bucket) == "rhmi-bucket" {
						return &s3.GetBucketTaggingOutput{
							TagSet: []*s3.Tag{
								{
									Key:   aws.String("integreatly.org/clusterID"),
									Value: aws.String("delete-me"),
								},
							}}, nil
					}
					return nil, nil
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &cleanupAwsAccountCmd{
				awsRegion:      "us-east-1",
				dryRun:         c.dryRun,
				clusterService: &clusterServiceMock{},
				ec2:            c.ec2,
				s3:             c.s3,
				s3Deleter:      &utils.MockS3BatchDeleter{BatchDeleteFunc: func(iterator s3manager.BatchDeleteIterator) error { return nil }},

				s3Buckets: []awsResourceObject{},
				vpcs:      []awsResourceObject{},

				osdResources:     map[string][]awsResourceObject{},
				rhmiResources:    map[string][]awsResourceObject{},
				deletedResources: []awsResourceObject{},
			}
			err := cmd.run(context.TODO())
			if err != nil && !c.expectError {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.expectedNumDeletedResources != len(cmd.deletedResources) {
				t.Fatalf("expect %d aws resources deleted but got %d", c.expectedNumDeletedResources, len(cmd.deletedResources))
			}
		})
	}
}

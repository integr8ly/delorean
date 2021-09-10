package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	loggingKeyBucket = "bucket-id"

	resourceTypeS3 = "s3"
)

var _ ClusterResourceManager = &S3Manager{}

type S3Manager struct {
	s3Client            s3Client
	s3BatchDeleteClient s3BatchDeleteClient
	taggingClient       taggingClient
	logger              *logrus.Entry
}

//s3Bucket internal representation of an s3 bucket containing only information required for reporting
type s3Bucket struct {
	ID  string
	ARN string
}

func NewDefaultS3Engine(session *session.Session, logger *logrus.Entry) *S3Manager {
	s3Client := s3.New(session)
	return &S3Manager{
		s3Client:            s3Client,
		s3BatchDeleteClient: s3manager.NewBatchDeleteWithClient(s3Client),
		taggingClient:       resourcegroupstaggingapi.New(session),
		logger:              logger.WithField(loggingKeyManager, managerS3),
	}
}

func (r *S3Manager) GetName() string {
	return "AWS S3 Manager"
}

func (s *S3Manager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	s.logger.Debug("delete s3 resources for cluster")
	//filter s3 buckets with correct tags
	s.logger.Debug("listing s3 buckets using provided tag filters")
	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeS3}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	getResourcesOutput, err := s.taggingClient.GetResources(getResourcesInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter s3 buckets in aws", s.logger)
	}
	//build list of s3 buckets to delete
	var bucketsToDelete []*s3Bucket
	for _, resourceTagMapping := range getResourcesOutput.ResourceTagMappingList {
		bucketARN := aws.StringValue(resourceTagMapping.ResourceARN)
		//get bucket id from arn, should be the last element
		//strings#Split will always return at least one element https://golang.org/pkg/strings/#Split
		bucketARNElements := strings.Split(bucketARN, ":")
		bucketID := bucketARNElements[len(bucketARNElements)-1]
		bucketsToDelete = append(bucketsToDelete, &s3Bucket{
			ID:  bucketID,
			ARN: bucketARN,
		})
	}
	s.logger.Debugf("found list of %d s3 buckets to delete", len(bucketsToDelete))
	//delete s3 buckets and build report
	var reportItems []*clusterservice.ReportItem
	for _, bucket := range bucketsToDelete {
		bucketLogger := s.logger.WithField(loggingKeyBucket, bucket.ID)
		bucketLogger.Debug("handling deletion for bucket")
		//add new item to report list for bucket
		reportItem := &clusterservice.ReportItem{
			ID:           bucket.ARN,
			Name:         bucket.ID,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		//don't delete in dry run scenario
		if dryRun {
			bucketLogger.Debug("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		//empty the bucket before performing the release
		bucketLogger.Debug("emptying all content from bucket before deletion")
		deleteIterator := s3manager.NewDeleteListIterator(s.s3Client, &s3.ListObjectsInput{
			Bucket: aws.String(bucket.ID),
		})
		if err := s.s3BatchDeleteClient.Delete(aws.BackgroundContext(), deleteIterator); err != nil {
			return nil, errors.WrapLog(err, "failed to empty bucket contents", bucketLogger)
		}
		//once the bucket is empty it can be deleted
		bucketLogger.Debug("performing bucket deletion")
		deleteBucketInput := &s3.DeleteBucketInput{
			Bucket: aws.String(bucket.ID),
		}
		if _, err := s.s3Client.DeleteBucket(deleteBucketInput); err != nil {
			return nil, errors.WrapLog(err, "failed to delete bucket", bucketLogger)
		}
	}
	//return final report
	return reportItems, nil
}

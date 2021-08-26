package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	resourceTypeElasticacheSnapshot = "elasticache:snapshot"
)

var _ ClusterResourceManager = &ElasticacheSnapshotManager{}

type ElasticacheSnapshotManager struct {
	elasticacheClient elasticacheiface.ElastiCacheAPI
	taggingClient     resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	logger            *logrus.Entry
}

func NewDefaultElasticacheSnapshotManager(session *session.Session, logger *logrus.Entry) *ElasticacheSnapshotManager {
	return &ElasticacheSnapshotManager{
		elasticacheClient: elasticache.New(session),
		taggingClient:     resourcegroupstaggingapi.New(session),
		logger:            logger.WithField(loggingKeyManager, managerElasticacheSnapshot),
	}
}

func (r *ElasticacheSnapshotManager) GetName() string {
	return "AWS ElastiCache Snapshot Manager"
}

func (r *ElasticacheSnapshotManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	logger := r.logger.WithFields(logrus.Fields{"clusterId": clusterId, "dryRun": dryRun})
	logger.Debug("deleting resources for cluster")

	//collection of clusterID's for respective snapshots
	var snapshotsToDelete []*basicResource

	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeElasticacheSnapshot}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	resourceOutput, err := r.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to get tagged snapshots", logger)
	}
	//convert response to standardised resource
	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		snapshotARN := aws.StringValue(resourceTagMapping.ResourceARN)
		snapshotARNElements := strings.Split(snapshotARN, ":")
		snapshotName := snapshotARNElements[len(snapshotARNElements)-1]
		snapshotLogger := r.logger.WithField(loggingKeySnapshot, snapshotName)
		describeSnapshotsOutput, err := r.elasticacheClient.DescribeSnapshots(&elasticache.DescribeSnapshotsInput{
			SnapshotName: aws.String(snapshotName),
		})
		if err != nil {
			return nil, errors.WrapLog(err, "failed to get elasticache snapshot", r.logger)
		}
		if len(describeSnapshotsOutput.Snapshots) == 0 {
			snapshotLogger.Debug("no snapshot found, assuming caching issue in aws, skipping")
			continue
		}
		snapshotsToDelete = append(snapshotsToDelete, &basicResource{
			Name: snapshotName,
			ARN:  snapshotARN,
		})
	}
	var reportItems []*clusterservice.ReportItem
	for _, snapshot := range snapshotsToDelete {
		snapshotLogger := r.logger.WithField(loggingKeySnapshot, snapshot.Name)
		snapshotLogger.Debug("handling deletion for snapshot")

		reportItem := &clusterservice.ReportItem{
			ID:           snapshot.ARN,
			Name:         snapshot.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			snapshotLogger.Debug("dry run is enabled, skipping")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		deleteSnapshotInput := &elasticache.DeleteSnapshotInput{
			SnapshotName: aws.String(snapshot.Name),
		}
		if _, err := r.elasticacheClient.DeleteSnapshot(deleteSnapshotInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == elasticache.ErrCodeInvalidSnapshotStateFault {
					snapshotLogger.Debug("snapshot is in a deleting state, ignoring error")
					continue
				}
				if awsErr.Code() == elasticache.ErrCodeSnapshotNotFoundFault {
					snapshotLogger.Debug("snapshot is not found, assuming already removed or aws caching, ignoring error")
					continue
				}
			}
			return nil, errors.WrapLog(err, "failed to delete snapshot", r.logger)
		}
	}
	return reportItems, nil
}

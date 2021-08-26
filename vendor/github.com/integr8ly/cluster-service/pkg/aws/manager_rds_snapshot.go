package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	loggingKeySnapshot = "snapshot-id"

	resourceTypeRDSSnapshot = "rds:snapshot"
)

type rdsSnapshot struct {
	ID  string
	ARN string
}

var _ ClusterResourceManager = &RDSSnapshotManager{}

type RDSSnapshotManager struct {
	rdsClient     rdsClient
	taggingClient taggingClient
	logger        *logrus.Entry
}

func NewDefaultRDSSnapshotManager(session *session.Session, logger *logrus.Entry) *RDSSnapshotManager {
	return &RDSSnapshotManager{
		rdsClient:     rds.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField(loggingKeyManager, managerRDSSnapshot),
	}
}

func (r *RDSSnapshotManager) GetName() string {
	return "AWS RDS Snapshot Manager"
}

func (r *RDSSnapshotManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	r.logger.Debug("delete snapshots for cluster")
	//filter with tags
	r.logger.Debug("listing rds snapshots using provided tag filters")
	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeRDSSnapshot}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	getResourcesOutput, err := r.taggingClient.GetResources(getResourcesInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter snapshots in aws", r.logger)
	}
	//build list of resources to delete
	var snapshotsToDelete []*rdsSnapshot
	for _, resourceTagMapping := range getResourcesOutput.ResourceTagMappingList {
		snapshotARN := aws.StringValue(resourceTagMapping.ResourceARN)
		//get resource id from arn, should be the last element
		//strings#Split will always return at least one element https://golang.org/pkg/strings/#Split
		snapshotARNElements := strings.Split(snapshotARN, ":")
		snapshotID := snapshotARNElements[len(snapshotARNElements)-1]
		snapshotsToDelete = append(snapshotsToDelete, &rdsSnapshot{
			ID:  snapshotID,
			ARN: snapshotARN,
		})
	}
	r.logger.Debugf("found list of %d rds snapshots to delete", len(snapshotsToDelete))
	//delete and build report
	var reportItems []*clusterservice.ReportItem
	for _, snapshot := range snapshotsToDelete {
		snapshotLogger := r.logger.WithField(loggingKeySnapshot, snapshot.ID)
		snapshotLogger.Debug("handling deletion for snapshot")
		//add new item to report list
		reportItem := &clusterservice.ReportItem{
			ID:           snapshot.ARN,
			Name:         snapshot.ID,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		//don't delete in dry run scenario
		if dryRun {
			snapshotLogger.Debug("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		describeSnapshotInput := &rds.DescribeDBSnapshotsInput{
			DBSnapshotIdentifier: aws.String(snapshot.ID),
		}
		describeSnapshotOutput, err := r.rdsClient.DescribeDBSnapshots(describeSnapshotInput)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == rds.ErrCodeDBSnapshotNotFoundFault {
				r.logger.Debug("snapshot not found, assuming it's been deleted")
				reportItem.ActionStatus = clusterservice.ActionStatusComplete
				continue
			}
			return nil, errors.WrapLog(err, "failed to describe db snapshots", r.logger)
		}
		if len(describeSnapshotOutput.DBSnapshots) != 1 {
			return nil, errors.WrapLog(err, "unexpected number of snapshots found", r.logger)
		}
		foundSnapshot := describeSnapshotOutput.DBSnapshots[0]
		if *foundSnapshot.SnapshotType != "manual" {
			r.logger.Debugf("unsupported snapshot type %s cannot be deleted, skipping", *foundSnapshot.SnapshotType)
			reportItem.ActionStatus = clusterservice.ActionStatusSkipped
			continue
		}
		if *foundSnapshot.Status != "available" {
			r.logger.Debugf("snapshot is not in an available state, current state is %s", *foundSnapshot.Status)
			reportItem.ActionStatus = clusterservice.ActionStatusSkipped
			continue

		}
		snapshotLogger.Debug("performing deletion request")
		deleteSnapshotInput := &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: aws.String(snapshot.ID),
		}
		if _, err := r.rdsClient.DeleteDBSnapshot(deleteSnapshotInput); err != nil {
			return nil, errors.WrapLog(err, "failed to delete rds snapshot", r.logger)
		}
	}
	//return final report
	return reportItems, nil
}

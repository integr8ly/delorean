package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	loggingKeyDatabase = "database-id"
)

var _ ClusterResourceManager = &RDSInstanceManager{}

type RDSInstanceManager struct {
	rdsClient rdsClient
	logger    *logrus.Entry
}

func NewDefaultRDSInstanceManager(session *session.Session, logger *logrus.Entry) *RDSInstanceManager {
	return &RDSInstanceManager{
		rdsClient: rds.New(session),
		logger:    logger.WithField("engine", managerRDS),
	}
}

func (r *RDSInstanceManager) GetName() string {
	return "AWS RDS Manager"
}

//Delete all RDS resources for a specified cluster
func (r *RDSInstanceManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	r.logger.Debug("deleting resources for cluster")
	clusterDescribeInput := &rds.DescribeDBInstancesInput{}
	clusterDescribeOutput, err := r.rdsClient.DescribeDBInstances(clusterDescribeInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to describe database clusters", r.logger)
	}
	var databasesToDelete []*rds.DBInstance
	for _, dbInstance := range clusterDescribeOutput.DBInstances {
		dbLogger := r.logger.WithField(loggingKeyDatabase, aws.StringValue(dbInstance.DBInstanceIdentifier))
		dbLogger.Debug("checking tags database cluster")
		tagListInput := &rds.ListTagsForResourceInput{
			ResourceName: dbInstance.DBInstanceArn,
		}
		tagListOutput, err := r.rdsClient.ListTagsForResource(tagListInput)
		if err != nil {
			return nil, errors.WrapLog(err, "failed to list tags for database cluster", dbLogger)
		}
		dbLogger.Debugf("checking for cluster tag match (%s=%s) on database", tagKeyClusterId, clusterId)
		if findTag(tagKeyClusterId, clusterId, tagListOutput.TagList) == nil {
			dbLogger.Debugf("database did not contain cluster tag match (%s=%s)", tagKeyClusterId, clusterId)
			continue
		}
		extraTagsMatch := true
		for extraTagKey, extraTagVal := range tags {
			dbLogger.Debugf("checking for additional tag match (%s=%s) on database", extraTagKey, extraTagVal)
			if findTag(extraTagKey, extraTagVal, tagListOutput.TagList) == nil {
				extraTagsMatch = false
				break
			}
		}
		if !extraTagsMatch {
			dbLogger.Debug("additional tags did not match, ignoring database")
			continue
		}
		databasesToDelete = append(databasesToDelete, dbInstance)
	}
	r.logger.Debugf("filtering complete, %d databases matched", len(databasesToDelete))
	reportItems := make([]*clusterservice.ReportItem, 0)
	for _, dbInstance := range databasesToDelete {
		dbLogger := r.logger.WithField(loggingKeyDatabase, aws.StringValue(dbInstance.DBInstanceIdentifier))
		dbLogger.Debugf("building report for database")
		reportItem := &clusterservice.ReportItem{
			ID:           aws.StringValue(dbInstance.DBInstanceArn),
			Name:         aws.StringValue(dbInstance.DBInstanceIdentifier),
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusEmpty,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			dbLogger.Debug("dry run enabled, skipping deletion step")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		dbLogger.Debug("performing deletion of database")
		reportItem.ActionStatus = clusterservice.ActionStatusInProgress
		//deleting will return an error if the database is already in a deleting state
		if aws.StringValue(dbInstance.DBInstanceStatus) == statusDeleting {
			dbLogger.Debugf("deletion of database already in progress")
			continue
		}
		if aws.BoolValue(dbInstance.DeletionProtection) {
			dbLogger.Debug("removing deletion protection on database")
			modifyInput := &rds.ModifyDBInstanceInput{
				DBInstanceIdentifier: dbInstance.DBInstanceIdentifier,
				DeletionProtection:   aws.Bool(false),
			}
			modifyOutput, err := r.rdsClient.ModifyDBInstance(modifyInput)
			if err != nil {
				return nil, errors.WrapLog(err, "failed to remove instance protection on database", dbLogger)
			}
			dbInstance = modifyOutput.DBInstance
		}

		deleteInput := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier:   dbInstance.DBInstanceIdentifier,
			DeleteAutomatedBackups: aws.Bool(true),
			SkipFinalSnapshot:      aws.Bool(true),
		}
		_, err := r.rdsClient.DeleteDBInstance(deleteInput)
		if err != nil {
			return nil, errors.WrapLog(err, "failed to delete rds instance", dbLogger)
		}
	}
	return reportItems, nil
}

func findTag(key, value string, tags []*rds.Tag) *rds.Tag {
	for _, tag := range tags {
		if key == aws.StringValue(tag.Key) && value == aws.StringValue(tag.Value) {
			return tag
		}
	}
	return nil
}

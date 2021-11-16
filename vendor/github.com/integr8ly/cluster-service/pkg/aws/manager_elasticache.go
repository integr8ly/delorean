package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ ClusterResourceManager = &ElasticacheManager{}

type ElasticacheManager struct {
	elasticacheClient    elasticacheiface.ElastiCacheAPI
	taggingClient        resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	logger               *logrus.Entry
	subnetGroupsToDelete []string
}

func NewDefaultElasticacheManager(session *session.Session, logger *logrus.Entry) *ElasticacheManager {
	return &ElasticacheManager{
		elasticacheClient:    elasticache.New(session),
		taggingClient:        resourcegroupstaggingapi.New(session),
		logger:               logger.WithField(loggingKeyManager, managerElasticache),
		subnetGroupsToDelete: make([]string, 0),
	}
}

func (r *ElasticacheManager) GetName() string {
	return "AWS ElastiCache Manager"
}

//Delete all elasticache resources for a specified cluster
func (r *ElasticacheManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	logger := r.logger.WithFields(logrus.Fields{"clusterId": clusterId, "dryRun": dryRun})
	logger.Debug("deleting resources for cluster")

	var reportItems []*clusterservice.ReportItem
	var replicationGroupsToDelete []string
	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{"elasticache:cluster"}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	resourceOutput, err := r.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to describe cache clusters", logger)
	}

	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnSplit := strings.Split(arn, ":")
		cacheClusterId := arnSplit[len(arnSplit)-1]
		cacheClusterInput := &elasticache.DescribeCacheClustersInput{
			CacheClusterId: aws.String(cacheClusterId),
		}
		cacheClusterOutput, err := r.elasticacheClient.DescribeCacheClusters(cacheClusterInput)
		if err != nil {
			return nil, errors.WrapLog(err, "cannot get cacheCluster output", logger)
		}
		for _, cacheCluster := range cacheClusterOutput.CacheClusters {
			rgLogger := logger.WithField("replicationGroup", cacheCluster.ReplicationGroupId)
			if contains(replicationGroupsToDelete, *cacheCluster.ReplicationGroupId) {
				rgLogger.Debugf("replication Group already exists in deletion list (%s=%s)", *cacheCluster.ReplicationGroupId, clusterId)
				break
			}
			replicationGroupsToDelete = append(replicationGroupsToDelete, *cacheCluster.ReplicationGroupId)
			// elasticache subnet groups don't support tags
			// add the cache subnet group to the subnetGroupsToDelete list
			// This way we can actually delete the subnet groups later on
			// when the caches are torn down
			r.subnetGroupsToDelete = appendIfUnique(r.subnetGroupsToDelete, *cacheCluster.CacheSubnetGroupName)
		}
	}
	logger.Debugf("filtering complete, %d replicationGroups matched", len(replicationGroupsToDelete))
	for _, replicationGroupId := range replicationGroupsToDelete {
		//delete each replication group in the list
		rgLogger := logger.WithField("replicationGroupId", aws.String(replicationGroupId))
		rgLogger.Debugf("building report for database")
		reportItem := &clusterservice.ReportItem{
			ID:           replicationGroupId,
			Name:         "elasticache Replication group",
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			rgLogger.Debug("dry run enabled, skipping deletion step")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		rgLogger.Debug("performing deletion of replication group")
		replicationGroupDescribeInput := &elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: &replicationGroupId,
		}
		replicationGroup, err := r.elasticacheClient.DescribeReplicationGroups(replicationGroupDescribeInput)
		if err != nil {
			return nil, errors.WrapLog(err, "cannot describe replicationGroups", logger)
		}
		//deleting will return an error if the replication group is already in a deleting state
		if len(replicationGroup.ReplicationGroups) > 0 &&
			aws.StringValue(replicationGroup.ReplicationGroups[0].Status) == statusDeleting {
			rgLogger.Debugf("deletion of replication Groups already in progress")
			reportItem.ActionStatus = clusterservice.ActionStatusInProgress
			continue
		}
		deleteReplicationGroupInput := &elasticache.DeleteReplicationGroupInput{
			ReplicationGroupId:   aws.String(replicationGroupId),
			RetainPrimaryCluster: aws.Bool(false),
		}
		if _, err := r.elasticacheClient.DeleteReplicationGroup(deleteReplicationGroupInput); err != nil {
			return nil, errors.WrapLog(err, "failed to delete elasticache replication group", logger)
		}
	}
	// handle deletion of orphaned cache subnet groups
	// elasticache subnet groups do not support tagging
	// which makes the logic a bit more tricky
	nextSubnetGroupsToDelete := make([]string, 0)

	for _, subnetGroupName := range r.subnetGroupsToDelete {
		sgLogger := logger.WithField("subnetGroup", aws.String(subnetGroupName))
		sgLogger.Debugf("building report for cache subnet groups")
		reportItem := &clusterservice.ReportItem{
			ID:           fmt.Sprintf("subnetgroup:%s", subnetGroupName),
			Name:         "elasticache subnet group",
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			sgLogger.Debug("dry run enabled, skipping deletion step")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		deleteSubnetGroupInput := &elasticache.DeleteCacheSubnetGroupInput{
			CacheSubnetGroupName: &subnetGroupName,
		}
		if _, err := r.elasticacheClient.DeleteCacheSubnetGroup(deleteSubnetGroupInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "CacheSubnetGroupInUse" {
				sgLogger.Debug("cache subnet group is still in use, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				// push the subnetGroup into the list of groups to be deleted next time
				nextSubnetGroupsToDelete = append(nextSubnetGroupsToDelete, subnetGroupName)
				continue
			}
			return nil, errors.WrapLog(err, "failed to delete cache subnet group", sgLogger)
		}
		reportItem.ActionStatus = clusterservice.ActionStatusComplete
	}
	r.subnetGroupsToDelete = nextSubnetGroupsToDelete
	if reportItems != nil {
		return reportItems, nil
	}
	return nil, nil
}

func contains(arr []string, targetValue string) bool {
	for _, element := range arr {
		if element != "" && element == targetValue {
			return true
		}
	}
	return false
}

func appendIfUnique(arr []string, targetValue string) []string {
	if !contains(arr, targetValue) {
		return append(arr, targetValue)
	}
	return arr
}

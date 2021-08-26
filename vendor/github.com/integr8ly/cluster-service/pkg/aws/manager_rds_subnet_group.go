package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	loggingKeySubnetGroup     = "subnet-group-name"
	resourceTypeDBSubnetGroup = "rds:subgrp"
)

var _ ClusterResourceManager = &RDSSubnetGroupManager{}

type RDSSubnetGroup struct {
	Name string
	ARN  string
}

type RDSSubnetGroupManager struct {
	rdsClient     rdsClient
	taggingClient taggingClient
	logger        *logrus.Entry
}

func NewDefaultRDSSubnetGroupManager(session *session.Session, logger *logrus.Entry) *RDSSubnetGroupManager {
	return &RDSSubnetGroupManager{
		rdsClient:     rds.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField("engine", managerRDS),
	}
}

func (r *RDSSubnetGroupManager) GetName() string {
	return "AWS RDS Subnet Group Manager"
}

// Delete all RDS Subnet Groups for a specified cluster
func (r *RDSSubnetGroupManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	r.logger.Debug("deleting resources for cluster")
	r.logger.Debug("listing rds subnet groups using provided tag filters")
	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeDBSubnetGroup}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	getResourcesOutput, err := r.taggingClient.GetResources(getResourcesInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter rds subnet groups", r.logger)
	}

	var subnetGroupsToDelete []*RDSSubnetGroup

	for _, resourceTagMapping := range getResourcesOutput.ResourceTagMappingList {
		subnetGroupARNElements := strings.Split(*resourceTagMapping.ResourceARN, ":")
		subnetGroupName := subnetGroupARNElements[len(subnetGroupARNElements)-1]

		subnetGroupsToDelete = append(subnetGroupsToDelete, &RDSSubnetGroup{
			Name: subnetGroupName,
			ARN:  *resourceTagMapping.ResourceARN,
		})
	}

	reportItems := make([]*clusterservice.ReportItem, 0)

	for _, dbSubnetGroup := range subnetGroupsToDelete {
		subnetGroupLogger := r.logger.WithField(loggingKeySubnetGroup, dbSubnetGroup.Name)
		subnetGroupLogger.Debug("creating report for rds subnet group")

		reportItem := &clusterservice.ReportItem{
			ID:           dbSubnetGroup.ARN,
			Name:         dbSubnetGroup.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusEmpty,
		}
		reportItems = append(reportItems, reportItem)

		if dryRun {
			subnetGroupLogger.Debug("dry run enabled, skipping deletion step")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		deleteInput := &rds.DeleteDBSubnetGroupInput{
			DBSubnetGroupName: aws.String(dbSubnetGroup.Name),
		}

		_, err := r.rdsClient.DeleteDBSubnetGroup(deleteInput)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidDBSubnetGroupStateFault" {
				subnetGroupLogger.Debug("the DB subnet group cannot be deleted because it's in use, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				continue
			}
			return nil, errors.WrapLog(err, "failed to delete rds db subnet group", subnetGroupLogger)
		}
		reportItem.ActionStatus = clusterservice.ActionStatusComplete
	}
	return reportItems, nil
}

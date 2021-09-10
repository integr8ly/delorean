package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	loggingKeyRouteTable = "route-table-id"

	resourceTypeRouteTable = "ec2:route-table"
)

var _ ClusterResourceManager = &RouteTableManager{}

type RouteTableManager struct {
	ec2Client     ec2Client
	taggingClient taggingClient
	logger        *logrus.Entry
}

func NewDefaultRouteTableManager(session *session.Session, logger *logrus.Entry) *RouteTableManager {
	return &RouteTableManager{
		ec2Client:     ec2.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField(loggingKeyManager, managerRouteTable),
	}
}

func (r *RouteTableManager) GetName() string {
	return "AWS EC2 RouteTable Manager"
}

func (r *RouteTableManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	r.logger.Debug("delete route table resources for cluster")
	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeRouteTable}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	resourceOutput, err := r.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter route tables", r.logger)
	}
	var routeTablesToDelete []*basicResource
	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnElements := strings.Split(arn, "/")
		routeTableId := arnElements[len(arnElements)-1]
		if routeTableId == "" {
			return nil, errors.WrapLog(err, fmt.Sprintf("invalid route table name from arn, %s", routeTableId), r.logger)
		}
		routeTablesToDelete = append(routeTablesToDelete, &basicResource{
			Name: routeTableId,
			ARN:  arn,
		})
	}
	r.logger.Debugf("found list of %d route tables to delete", len(routeTablesToDelete))
	//delete resources
	var reportItems []*clusterservice.ReportItem
	for _, routeTable := range routeTablesToDelete {
		routeTableLogger := r.logger.WithField(loggingKeyRouteTable, routeTable.Name)
		reportItem := &clusterservice.ReportItem{
			ID:           routeTable.ARN,
			Name:         routeTable.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			routeTableLogger.Debugf("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		routeTableLogger.Debugf("performing route table deletion")
		deleteRouteTableInput := &ec2.DeleteRouteTableInput{
			RouteTableId: aws.String(routeTable.Name),
		}
		if _, err := r.ec2Client.DeleteRouteTable(deleteRouteTableInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DependencyViolation" {
				routeTableLogger.Debug("route table has existing dependencies which have not been deleted, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				continue
			}

			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidRouteTableID.NotFound" {
				routeTableLogger.Debug("route does not exist, assuming deleted")
				reportItem.ActionStatus = clusterservice.ActionStatusComplete
				continue
			}
			return nil, errors.WrapLog(err, "failed to delete route table", r.logger)
		}
		reportItem.ActionStatus = clusterservice.ActionStatusComplete
	}
	return reportItems, nil
}

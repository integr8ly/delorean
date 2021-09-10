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
	loggingKeyVpcPeeringConnection = "vpc-peering-id"

	resourceTypeVpcPeeringConnection = "ec2:vpc-peering-connection"
)

var _ ClusterResourceManager = &VpcPeeringManager{}

// VpcPeeringManager type
type VpcPeeringManager struct {
	ec2Client     ec2Client
	taggingClient taggingClient
	logger        *logrus.Entry
}

// NewDefaultVpcPeeringManager create session for manager
func NewDefaultVpcPeeringManager(session *session.Session, logger *logrus.Entry) *VpcPeeringManager {
	return &VpcPeeringManager{
		ec2Client:     ec2.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField(loggingKeyManager, managerVpcPeering),
	}
}

// GetName getter function
func (r *VpcPeeringManager) GetName() string {
	return "AWS EC2 Vpc Peering Connection Manager"
}

// DeleteResourcesForCluster deletes resource for cluster
func (r *VpcPeeringManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	var vpcPeeringConnectionsToDelete []*basicResource
	r.logger.Debug("delete vpc peering connections resources for cluster")
	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeVpcPeeringConnection}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	// add to the vpc peering delete array
	resourceOutput, err := r.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter vpc peering connections", r.logger)
	}

	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnElements := strings.Split(arn, "/")
		vpcPeeringConnectionID := arnElements[len(arnElements)-1]
		if vpcPeeringConnectionID == "" {
			return nil, errors.WrapLog(err, fmt.Sprintf("invalid vpc peering connection name from arn, %s", vpcPeeringConnectionID), r.logger)
		}
		vpcPeeringConnectionsToDelete = append(vpcPeeringConnectionsToDelete, &basicResource{
			Name: vpcPeeringConnectionID,
			ARN:  arn,
		})
		r.logger.Debugf("found list of %d vpc peering connection to delete", len(vpcPeeringConnectionsToDelete))
	}
	//delete resources
	var reportItems []*clusterservice.ReportItem
	for _, vpcPeeringConnection := range vpcPeeringConnectionsToDelete {
		vpcPeeringConnectionLogger := r.logger.WithField(loggingKeyVpcPeeringConnection, vpcPeeringConnection.Name)
		reportItem := &clusterservice.ReportItem{
			ID:           vpcPeeringConnection.ARN,
			Name:         vpcPeeringConnection.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			vpcPeeringConnectionLogger.Debugf("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		vpcPeeringConnectionLogger.Debugf("performing vpc peering connection deletion")
		deleteVpcPeeringConnectionInput := &ec2.DeleteVpcPeeringConnectionInput{
			VpcPeeringConnectionId: aws.String(vpcPeeringConnection.Name),
		}

		if _, err := r.ec2Client.DeleteVpcPeeringConnection(deleteVpcPeeringConnectionInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DependencyViolation" {
				vpcPeeringConnectionLogger.Debug("vpc peering connection has existing dependencies which have not been deleted, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				r.logger.Infof("Error: %s, %s", awsErr.Code(), awsErr.Message())
				continue
			}

			// in the case of vpc peerings they are picked up on describe but then when attempting to delete they are not found
			// any that are not found can be skipped.
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidVpcPeeringConnectionID.NotFound" {
				vpcPeeringConnectionLogger.Debug("vpc peering connection does not exist, assume deleted")
				reportItem.ActionStatus = clusterservice.ActionStatusComplete
				continue
			}

			return nil, errors.WrapLog(err, "failed to delete vpc peering connection", r.logger)
		}
		reportItem.ActionStatus = clusterservice.ActionStatusComplete
	}
	return reportItems, nil
}

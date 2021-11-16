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
	loggingKeyVpc = "vpc-id"

	resourceTypeVpc = "ec2:vpc"
)

var _ ClusterResourceManager = &VpcManager{}

// VpcManager type
type VpcManager struct {
	ec2Client     ec2Client
	taggingClient taggingClient
	logger        *logrus.Entry
}

// NewDefaultVpcManager create session for manager
func NewDefaultVpcManager(session *session.Session, logger *logrus.Entry) *VpcManager {
	return &VpcManager{
		ec2Client:     ec2.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField(loggingKeyManager, managerVpc),
	}
}

// GetName getter function
func (r *VpcManager) GetName() string {
	return "AWS EC2 Vpc Manager"
}

// DeleteResourcesForCluster deletes resource for cluster
func (r *VpcManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	var vpcsToDelete []*basicResource
	r.logger.Debug("delete vpc resources for cluster")
	//  integreatly.org/clusterID tags
	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeVpc}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	// add to the vpc delete array
	resourceOutput, err := r.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter vpc", r.logger)
	}
	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnElements := strings.Split(arn, "/")
		vpcID := arnElements[len(arnElements)-1]
		if vpcID == "" {
			return nil, errors.WrapLog(err, fmt.Sprintf("invalid vpc name from arn, %s", vpcID), r.logger)
		}
		vpcsToDelete = append(vpcsToDelete, &basicResource{
			Name: vpcID,
			ARN:  arn,
		})
	}

	r.logger.Debugf("found list of %d vpc to delete", len(vpcsToDelete))
	//delete resources
	var reportItems []*clusterservice.ReportItem
	for _, vpc := range vpcsToDelete {
		vpcLogger := r.logger.WithField(loggingKeyVpc, vpc.Name)
		reportItem := &clusterservice.ReportItem{
			ID:           vpc.ARN,
			Name:         vpc.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			vpcLogger.Debugf("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		vpcLogger.Debugf("performing vpc deletion")
		deleteVpcInput := &ec2.DeleteVpcInput{
			VpcId: aws.String(vpc.Name),
		}
		if _, err := r.ec2Client.DeleteVpc(deleteVpcInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DependencyViolation" {
				vpcLogger.Debug("vpc has existing dependencies which have not been deleted, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				continue
			}
			return nil, errors.WrapLog(err, "failed to delete vpc", r.logger)
		}
	}
	return reportItems, nil
}

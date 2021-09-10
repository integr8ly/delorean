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
	loggingKeySecurityGroup = "security-group-id"

	resourceTypeSecurtyGroup = "ec2:security-group"
)

var _ ClusterResourceManager = &SecurityGroupManager{}

// SecurityGroupManager type
type SecurityGroupManager struct {
	ec2Client     ec2Client
	taggingClient taggingClient
	logger        *logrus.Entry
}

// NewDefaultSecurityGroupManager create session for manager
func NewDefaultSecurityGroupManager(session *session.Session, logger *logrus.Entry) *SecurityGroupManager {
	return &SecurityGroupManager{
		ec2Client:     ec2.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField(loggingKeyManager, managerSecurityGroup),
	}
}

// GetName getter function
func (r *SecurityGroupManager) GetName() string {
	return "AWS EC2 SecurityGroup Manager"
}

// DeleteResourcesForCluster deletes resource for cluster
func (r *SecurityGroupManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	var securityGroupsToDelete []*basicResource
	r.logger.Debug("delete security groups resources for cluster")
	//  integreatly.org/clusterID tags
	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeSecurtyGroup}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	// add to the security group delete array
	resourceOutput, err := r.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter security groups", r.logger)
	}

	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnElements := strings.Split(arn, "/")
		securityGroupID := arnElements[len(arnElements)-1]
		if securityGroupID == "" {
			return nil, errors.WrapLog(err, fmt.Sprintf("invalid security groups name from arn, %s", securityGroupID), r.logger)
		}
		securityGroupsToDelete = append(securityGroupsToDelete, &basicResource{
			Name: securityGroupID,
			ARN:  arn,
		})
		r.logger.Debugf("found list of %d security groups to delete", len(securityGroupsToDelete))
	}
	//delete resources
	var reportItems []*clusterservice.ReportItem
	for _, securityGroup := range securityGroupsToDelete {
		securityGroupLogger := r.logger.WithField(loggingKeySecurityGroup, securityGroup.Name)
		reportItem := &clusterservice.ReportItem{
			ID:           securityGroup.ARN,
			Name:         securityGroup.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			securityGroupLogger.Debugf("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		securityGroupLogger.Debugf("performing security group deletion")
		deleteSecurityGroupInput := &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(securityGroup.Name),
		}
		if _, err := r.ec2Client.DeleteSecurityGroup(deleteSecurityGroupInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DependencyViolation" {
				securityGroupLogger.Debug("security group has existing dependencies which have not been deleted, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				continue
			}
			return nil, errors.WrapLog(err, "failed to delete security group", r.logger)
		}
		reportItem.ActionStatus = clusterservice.ActionStatusComplete
	}
	return reportItems, nil
}

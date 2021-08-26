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
	loggingKeySubnet = "subnet-id"

	resourceTypeSubnet = "ec2:subnet"
)

var _ ClusterResourceManager = &SubnetManager{}

type SubnetManager struct {
	ec2Client     ec2Client
	taggingClient taggingClient
	logger        *logrus.Entry
}

func NewDefaultSubnetManager(session *session.Session, logger *logrus.Entry) *SubnetManager {
	return &SubnetManager{
		ec2Client:     ec2.New(session),
		taggingClient: resourcegroupstaggingapi.New(session),
		logger:        logger.WithField(loggingKeyManager, managerSubnet),
	}
}

func (r *SubnetManager) GetName() string {
	return "AWS EC2 Subnet Manager"
}

func (s *SubnetManager) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error) {
	s.logger.Debug("delete subnet resources for cluster")
	resourceInput := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{resourceTypeSubnet}),
		TagFilters:          convertClusterTagsToAWSTagFilter(clusterId, tags),
	}
	resourceOutput, err := s.taggingClient.GetResources(resourceInput)
	if err != nil {
		return nil, errors.WrapLog(err, "failed to filter subnets", s.logger)
	}
	var subnetsToDelete []*basicResource
	for _, resourceTagMapping := range resourceOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnElements := strings.Split(arn, "/")
		subnetId := arnElements[len(arnElements)-1]
		if subnetId == "" {
			return nil, errors.WrapLog(err, fmt.Sprintf("invalid subnet name from arn, %s", subnetId), s.logger)
		}
		subnetsToDelete = append(subnetsToDelete, &basicResource{
			Name: subnetId,
			ARN:  arn,
		})
	}
	s.logger.Debugf("found list of %d subnets to delete", len(subnetsToDelete))
	//delete resources
	var reportItems []*clusterservice.ReportItem
	for _, subnet := range subnetsToDelete {
		subnetLogger := s.logger.WithField(loggingKeySubnet, subnet.Name)
		reportItem := &clusterservice.ReportItem{
			ID:           subnet.ARN,
			Name:         subnet.Name,
			Action:       clusterservice.ActionDelete,
			ActionStatus: clusterservice.ActionStatusInProgress,
		}
		reportItems = append(reportItems, reportItem)
		if dryRun {
			subnetLogger.Debugf("dry run is enabled, skipping deletion")
			reportItem.ActionStatus = clusterservice.ActionStatusDryRun
			continue
		}
		subnetLogger.Debugf("performing subnet deletion")
		deleteSubnetInput := &ec2.DeleteSubnetInput{
			SubnetId: aws.String(subnet.Name),
		}
		if _, err := s.ec2Client.DeleteSubnet(deleteSubnetInput); err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DependencyViolation" {
				subnetLogger.Debug("subnet has existing dependencies which have not been deleted, skipping")
				reportItem.ActionStatus = clusterservice.ActionStatusSkipped
				continue
			}
			return nil, errors.WrapLog(err, "failed to delete subnet", s.logger)
		}
		reportItem.ActionStatus = clusterservice.ActionStatusComplete
	}

	return reportItems, nil
}

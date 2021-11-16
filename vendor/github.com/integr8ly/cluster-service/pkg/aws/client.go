package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
	"github.com/integr8ly/cluster-service/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ clusterservice.Client = &Client{}

type Client struct {
	ResourceManagers []ClusterResourceManager
	Logger           *logrus.Entry
}

func NewDefaultClient(awsSession *session.Session, logger *logrus.Entry) *Client {
	log := logger.WithField("cluster_service_provider", "aws")
	rdsManager := NewDefaultRDSInstanceManager(awsSession, logger)
	rdsSnapshotManager := NewDefaultRDSSnapshotManager(awsSession, logger)
	rdsSubnetGroupManager := NewDefaultRDSSubnetGroupManager(awsSession, logger)
	s3Manager := NewDefaultS3Engine(awsSession, logger)
	elasticacheManager := NewDefaultElasticacheManager(awsSession, logger)
	elasticacheSnapshotManager := NewDefaultElasticacheSnapshotManager(awsSession, logger)
	vpcPeeringManager := NewDefaultVpcPeeringManager(awsSession, logger)
	subnetManager := NewDefaultSubnetManager(awsSession, logger)
	securityGroupManager := NewDefaultSecurityGroupManager(awsSession, logger)
	routeTableManager := NewDefaultRouteTableManager(awsSession, logger)
	vpcManager := NewDefaultVpcManager(awsSession, logger)
	return &Client{
		ResourceManagers: []ClusterResourceManager{rdsManager, rdsSubnetGroupManager, elasticacheManager, s3Manager, rdsSnapshotManager, elasticacheSnapshotManager, vpcPeeringManager, subnetManager, securityGroupManager, routeTableManager, vpcManager},
		Logger:           log,
	}
}

//DeleteResourcesForCluster Delete AWS resources based on tags using provided action engines
func (c *Client) DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) (*clusterservice.Report, error) {
	logger := c.Logger.WithFields(logrus.Fields{loggingKeyClusterID: clusterId, loggingKeyDryRun: dryRun})
	logger.Debugf("deleting resources for cluster")
	report := &clusterservice.Report{}
	for _, engine := range c.ResourceManagers {
		engineLogger := logger.WithField(loggingKeyManager, engine.GetName())
		engineLogger.Debugf("found Logger")
		reportItems, err := engine.DeleteResourcesForCluster(clusterId, tags, dryRun)
		if err != nil {
			return nil, errors.WrapLog(err, fmt.Sprintf("failed to run engine %s", engine.GetName()), engineLogger)
		}
		report.Items = append(report.Items, reportItems...)
	}
	return report, nil
}

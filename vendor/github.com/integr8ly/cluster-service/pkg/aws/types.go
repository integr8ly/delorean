package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/cluster-service/pkg/clusterservice"
)

type ResourceManagerType string

const (
	tagKeyClusterId = "integreatly.org/clusterID"
	statusDeleting  = "deleting"

	managerRDS                 ResourceManagerType = "aws_rds"
	managerS3                  ResourceManagerType = "aws_s3"
	managerSubnet              ResourceManagerType = "aws_ec2_subnet"
	managerVpc                 ResourceManagerType = "aws_ec2_vpc"
	managerVpcPeering          ResourceManagerType = "aws_ec2_vpc_peering"
	managerRDSSnapshot         ResourceManagerType = "aws_rds_snapshot"
	managerElasticache         ResourceManagerType = "aws_elasticache"
	managerElasticacheSnapshot ResourceManagerType = "aws_elasticache_snapshot"
	managerSecurityGroup       ResourceManagerType = "aws_ec2_security_group"
	managerRouteTable          ResourceManagerType = "aws_ec2_route_table"

	loggingKeyClusterID = "cluster-id"
	loggingKeyDryRun    = "dry-run"
	loggingKeyManager   = "manager"
)

//go:generate moq -out moq_crm_test.go . ClusterResourceManager
//ClusterResourceManager Perform actions for a specific resource
type ClusterResourceManager interface {
	GetName() string
	DeleteResourcesForCluster(clusterId string, tags map[string]string, dryRun bool) ([]*clusterservice.ReportItem, error)
}

//basicResource Representation of basic AWS resource information
type basicResource struct {
	Name string
	ARN  string
}

//go:generate moq -out moq_rdsclient_test.go . rdsClient
//rdsClient alias for use with moq
type rdsClient interface {
	rdsiface.RDSAPI
}

//go:generate moq -out moq_elasticacheclient_test.go . elasticacheClient
//elasticacheClient alias for use with moq
type elasticacheClient interface {
	elasticacheiface.ElastiCacheAPI
}

//go:generate moq -out moq_s3client_test.go . s3Client
//s3Client alias for use with moq
type s3Client interface {
	s3iface.S3API
}

//go:generate moq -out moq_s3batchdeleteclient.go . s3BatchDeleteClient
//s3BatchDeleteClient alias for use with moq
type s3BatchDeleteClient interface {
	s3manageriface.BatchDelete
}

//go:generate moq -out moq_ec2client_test.go . ec2Client
//ec2Client alias for use with moq
type ec2Client interface {
	ec2iface.EC2API
}

//go:generate moq -out moq_taggingclient_test.go . taggingClient
//taggingClient alias for use with moq
type taggingClient interface {
	resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}

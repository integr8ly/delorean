package cmd

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/spf13/cobra"

	clusterServiceAws "github.com/integr8ly/cluster-service/pkg/aws"
)

const (
	veleroS3BucketPrefix  = "managed-velero"
	osdClusterTagPrefix   = "kubernetes.io/cluster/"
	rhmiResourceTagPrefix = "integreatly.org/clusterID"
)

type cleanupAwsAccountCmd struct {
	awsRegion          string
	deleteUntaggedVpcs bool
	dryRun             bool

	clusterService clusterServiceAws.Client
	ec2            ec2.EC2
	elasticache    elasticache.ElastiCache
	rds            rds.RDS
	s3             s3iface.S3API
	s3Deleter      s3manageriface.BatchDelete

	osdResources  map[string][]awsResourceObject
	rhmiResources map[string][]awsResourceObject
	s3Buckets     []awsResourceObject
	vpcs          []awsResourceObject
}

type cleanupAwsAccountCmdFlags struct {
	debug              bool
	deleteUntaggedVpcs bool
	dryRun             bool
	awsRegion          string
}

type awsResourceObject struct {
	ID           string
	tags         map[string]string
	resourceType string
}

func init() {
	f := &cleanupAwsAccountCmdFlags{}
	cmd := &cobra.Command{
		Use:   "cleanup-aws",
		Short: "Clean up AWS account. Remove unused s3 buckets (cluster backups), empty VPCs and all RHMI resources left behind",
		Run: func(cmd *cobra.Command, args []string) {

			c, err := newcleanupAwsAccountCmd(f)
			if err != nil {
				handleError(err)
			}
			if err := c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}
	pipelineCmd.AddCommand(cmd)
	cmd.Flags().StringVar(&f.awsRegion, "region", "", "AWS region to cleanup")
	if err := cmd.MarkFlagRequired("region"); err != nil {
		handleError(err)
	}
	cmd.Flags().BoolVar(&f.deleteUntaggedVpcs, "delete-untagged-vpcs", false, "If true, delete untagged VPCs")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", true, "If true, only list the resources that will be deleted")
	cmd.Flags().BoolVar(&f.debug, "debug", false, "Enable debug mode")
}

func newcleanupAwsAccountCmd(f *cleanupAwsAccountCmdFlags) (*cleanupAwsAccountCmd, error) {
	awsKeyId, err := requireValue("AWS_ACCESS_KEY_ID")
	if err != nil {
		handleError(err)
	}
	awsSecretKey, err := requireValue("AWS_SECRET_ACCESS_KEY")
	if err != nil {
		handleError(err)
	}
	awsSession := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsKeyId, awsSecretKey, ""),
		Region:      aws.String(f.awsRegion),
	}))

	if f.debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Debug("creating a new ec2 session")
	ec2Session := ec2.New(awsSession)
	log.Debug("creating a new s3 session")
	s3Session := s3.New(awsSession)
	s3Deleter := s3manager.NewBatchDelete(awsSession)

	log.Debug("creating a new cluster-service session")
	cs := clusterServiceAws.NewDefaultClient(awsSession, log.WithField("service", "cluster_service"))

	return &cleanupAwsAccountCmd{
		awsRegion:          f.awsRegion,
		deleteUntaggedVpcs: f.deleteUntaggedVpcs,
		dryRun:             f.dryRun,
		clusterService:     *cs,
		ec2:                *ec2Session,
		s3:                 s3Session,
		s3Deleter:          s3Deleter,

		s3Buckets:     []awsResourceObject{},
		vpcs:          []awsResourceObject{},
		osdResources:  map[string][]awsResourceObject{},
		rhmiResources: map[string][]awsResourceObject{},
	}, nil
}

func (c *cleanupAwsAccountCmd) run(ctx context.Context) error {
	if c.dryRun {
		log.Info("DRY RUN (No AWS resources will be deleted)")
	}

	if err := c.fetchVpcs(ctx); err != nil {
		return err
	}

	if err := c.fetchS3Buckets(ctx); err != nil {
		return err
	}

	if err := c.analyzeAwsResourceTags(c.s3Buckets); err != nil {
		return err
	}

	if err := c.analyzeAwsResourceTags(c.vpcs); err != nil {
		return err
	}

	if err := c.cleanupUnusedVeleroS3Buckets(ctx); err != nil {
		return err
	}

	if err := c.cleanupEmptyVpcs(); err != nil {
		return err
	}

	if len(c.rhmiResources) > 0 {
		for tag, awsObject := range c.rhmiResources {
			// If RHMI/RHOAM resource is associated with an OSD cluster resource, then skip the deletion
			if _, ok := c.osdResources[tag]; ok {
				c.osdResources[tag] = append(c.osdResources[tag], awsObject...)
			} else {
				if c.dryRun {
					log.Info("would use cluster-service to delete resources with a tag:", tag)
				} else {
					report, err := c.clusterService.DeleteResourcesForCluster(tag, map[string]string{}, c.dryRun)

					if err != nil {
						log.Debug("error when running clusterService", err)
						return err
					}

					if len(report.Items) < 1 {
						log.Debug("no items to delete")
					}

					for _, item := range report.Items {
						log.Info(item.Name, item.ID, item.Action, item.ActionStatus)
					}
				}
			}

		}
	}

	log.Info("Following OSD cluster resources were not deleted")
	for _, awsResources := range c.osdResources {
		for _, j := range awsResources {
			log.Infof("Resource type: %s, ID: %s, Tags: %+v\n", j.resourceType, j.ID, j.tags)
		}
	}

	return nil
}

func (c *cleanupAwsAccountCmd) fetchS3Buckets(ctx context.Context) error {
	log.Debug("Fetch S3 Buckets")

	fetchedBuckets, err := c.s3.ListBucketsWithContext(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("failed to list s3 buckets: %v", err)
	}

	for _, b := range fetchedBuckets.Buckets {
		bucketLocation, err := c.s3.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: b.Name})
		if err != nil {
			log.Infof("failed to get s3 bucket location (ignoring): %v \n", err)
			continue
		}

		bucketRegion := "us-east-1"
		if bucketLocation.LocationConstraint != nil {
			bucketRegion = *bucketLocation.LocationConstraint
		}

		if bucketRegion == c.awsRegion {
			tagging, err := c.s3.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String(*b.Name)})
			if err != nil {
				log.Warnf("failed to get s3 bucket tags: %v \n", err)
				continue
			}

			newS3Bucket := awsResourceObject{
				ID:           *b.Name,
				resourceType: "s3",
				tags:         map[string]string{},
			}

			for _, tag := range tagging.TagSet {
				newS3Bucket.tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
			}

			c.s3Buckets = append(c.s3Buckets, newS3Bucket)
		}

	}
	return nil
}

func (c *cleanupAwsAccountCmd) fetchVpcs(ctx context.Context) error {
	log.Debug("Fetching VPCs")

	fetchedVpcs, err := c.ec2.DescribeVpcsWithContext(ctx, nil)
	if err != nil {
		return err
	}

	for _, vpc := range fetchedVpcs.Vpcs {
		// Skip default VPCs
		if *vpc.IsDefault {
			continue
		}
		newVpc := awsResourceObject{
			ID:           aws.StringValue(vpc.VpcId),
			resourceType: "vpc",
			tags:         map[string]string{},
		}
		for _, tag := range vpc.Tags {
			newVpc.tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
		}
		c.vpcs = append(c.vpcs, newVpc)
	}
	return nil
}

func (c *cleanupAwsAccountCmd) analyzeAwsResourceTags(awsObjects []awsResourceObject) error {

	for _, obj := range awsObjects {
		for key, value := range obj.tags {
			if strings.Contains(key, osdClusterTagPrefix) {
				clusterTag := strings.Replace(key, osdClusterTagPrefix, "", 1)
				c.osdResources[clusterTag] = append(c.osdResources[clusterTag], obj)
			} else if strings.Contains(key, rhmiResourceTagPrefix) {
				c.rhmiResources[value] = append(c.rhmiResources[value], obj)
			}
		}
	}

	return nil

}

func (c *cleanupAwsAccountCmd) cleanupUnusedVeleroS3Buckets(ctx context.Context) error {
	log.Debug("Deleting S3 Buckets")
	for _, b := range c.s3Buckets {
		if isVeleroBucket(b.ID) {
			if !c.containsActiveOsdClusterTag(b.tags) {
				if c.dryRun {
					log.Info("would delete a bucket: ", b.ID)
				} else {
					if err := c.removeS3Bucket(b.ID, ctx); err != nil {
						return err
					}
					log.Infof("s3 bucket %s successfully deleted\n", b.ID)
				}
			}

		}
	}
	return nil
}

func (c *cleanupAwsAccountCmd) cleanupEmptyVpcs() error {
	log.Debug("Deleting Empty VPCs")
	for _, vpc := range c.vpcs {
		if len(vpc.tags) < 1 {
			if c.dryRun {
				log.Info("would delete a vpc: ", vpc.ID)
			} else {
				_, err := c.ec2.DeleteVpc(&ec2.DeleteVpcInput{
					VpcId: aws.String(vpc.ID),
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func isVeleroBucket(bucketName string) bool {
	if strings.Contains(bucketName, veleroS3BucketPrefix) {
		return true
	}
	return false
}

func (c *cleanupAwsAccountCmd) containsActiveOsdClusterTag(tags map[string]string) bool {
	for _, value := range tags {
		if _, ok := c.osdResources[value]; ok {
			return true
		}
	}
	return false
}

func (c *cleanupAwsAccountCmd) removeS3Bucket(bucketName string, ctx context.Context) error {
	bucketObjects, err := c.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return err
	}

	if err := c.deleteObjects(ctx, bucketName, bucketObjects.Contents); err != nil {
		return err
	}

	_, err = c.s3.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(bucketName)})
	if err != nil {
		return err
	}
	return nil
}

func (c *cleanupAwsAccountCmd) deleteObjects(ctx context.Context, bucket string, toDelete []*s3.Object) error {
	if len(toDelete) == 0 {
		log.Debug(fmt.Sprintf("[%s] No objects to delete", bucket))
		return nil
	}
	log.Debug(fmt.Sprintf("[%s] Deleting %d objects", bucket, len(toDelete)))
	var batch []s3manager.BatchDeleteObject
	for _, o := range toDelete {
		b := s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Key:    o.Key,
				Bucket: aws.String(bucket),
			},
		}
		batch = append(batch, b)
	}
	if err := c.s3Deleter.Delete(ctx, &s3manager.DeleteObjectsIterator{Objects: batch}); err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("[%s] %d objects deleted", bucket, len(toDelete)))
	return nil
}

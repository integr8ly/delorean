package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	clusterServiceAws "github.com/integr8ly/cluster-service/pkg/aws"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/spf13/cobra"

	clusterService "github.com/integr8ly/cluster-service/pkg/clusterservice"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	managedVeleroTagPrefix = "velero.io/infrastructureName"
	veleroS3BucketPrefix   = "managed-velero"
	osdClusterTagPrefix    = "kubernetes.io/cluster/"
	rhmiResourceTagPrefix  = "integreatly.org/clusterID"
)

type cleanupAwsAccountCmd struct {
	awsRegion string
	dryRun    bool

	clusterService clusterService.Client
	ec2            ec2iface.EC2API
	s3             s3iface.S3API
	s3Deleter      s3manageriface.BatchDelete

	osdResources     map[string][]awsResourceObject
	rhmiResources    map[string][]awsResourceObject
	deletedResources []awsResourceObject
	s3Buckets        []awsResourceObject
	vpcs             []awsResourceObject
}

type cleanupAwsAccountCmdFlags struct {
	debug     bool
	dryRun    bool
	awsRegion string
}

type awsResourceObject struct {
	ID           string
	tags         map[string]string
	clusterTag   string
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
		awsRegion:      f.awsRegion,
		dryRun:         f.dryRun,
		clusterService: cs,
		ec2:            ec2Session,
		s3:             s3Session,
		s3Deleter:      s3Deleter,

		s3Buckets:        []awsResourceObject{},
		vpcs:             []awsResourceObject{},
		osdResources:     map[string][]awsResourceObject{},
		rhmiResources:    map[string][]awsResourceObject{},
		deletedResources: []awsResourceObject{},
	}, nil
}

func (c *cleanupAwsAccountCmd) run(ctx context.Context) error {
	if c.dryRun {
		log.Info("DRY RUN (No AWS resources will be deleted)")
	}

	if err := c.fetchEC2Instances(ctx); err != nil {
		return err
	}
	if err := c.fetchVpcs(ctx); err != nil {
		return err
	}
	if err := c.fetchS3Buckets(ctx); err != nil {
		return err
	}

	if len(c.rhmiResources) > 0 {
		for tag, awsObjects := range c.rhmiResources {
			// If RHMI/RHOAM resource tag is associated with an OSD cluster resource, then skip the deletion
			// and add the resource to the list of OSD cluster resources that won't be deleted
			// Else delete it with cluster-service
			if _, ok := c.osdResources[tag]; ok {
				c.osdResources[tag] = append(c.osdResources[tag], awsObjects...)
			} else {
				if c.dryRun {
					log.Info("Would use cluster-service to delete resources with a tag: ", tag)
				} else {
					if err := c.cleanupWithClusterService(tag); err != nil {
						return err
					}
					c.deletedResources = append(c.deletedResources, awsObjects...)
				}
			}

		}
	}

	if err := c.cleanupUnusedVeleroS3Buckets(ctx); err != nil {
		return err
	}

	if err := c.cleanupVpcs(); err != nil {
		return err
	}

	if len(c.osdResources) != 0 {
		log.Info("Following OSD cluster resources won't be deleted")
		for _, awsResources := range c.osdResources {
			for _, j := range awsResources {
				log.Infof("Resource type: %s, ID: %s, Cluster tag: %+v\n", j.resourceType, j.ID, j.clusterTag)
			}
		}
	}

	log.Debugf("Deleted resources: %+v\n", c.deletedResources)

	return nil
}

func (c *cleanupAwsAccountCmd) fetchS3Buckets(ctx context.Context) error {
	log.Debug("Fetching S3 Buckets")

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
			tagging, err := c.s3.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: b.Name})
			if err != nil {
				log.Warnf("failed to get s3 bucket tags: %v \n", err)
				log.Debug(err)
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
			var hasRhmiTag bool
			newS3Bucket.clusterTag, hasRhmiTag = extractClusterTag(newS3Bucket.tags)

			if hasRhmiTag {
				c.rhmiResources[newS3Bucket.clusterTag] = append(c.rhmiResources[newS3Bucket.clusterTag], newS3Bucket)
			} else {
				c.s3Buckets = append(c.s3Buckets, newS3Bucket)
			}

		}

	}
	return nil
}

func (c *cleanupAwsAccountCmd) fetchEC2Instances(ctx context.Context) error {
	log.Debug("Fetching EC2 Instances")

	fetchedReservations, err := c.ec2.DescribeInstancesWithContext(ctx, nil)
	if err != nil {
		return err
	}

	for _, reservation := range fetchedReservations.Reservations {
		for _, ec2Instance := range reservation.Instances {
			newEc2 := awsResourceObject{
				ID:           aws.StringValue(ec2Instance.InstanceId),
				resourceType: "ec2Instance",
				tags:         map[string]string{},
			}
			for _, tag := range ec2Instance.Tags {
				newEc2.tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
			}
			newEc2.clusterTag, _ = extractClusterTag(newEc2.tags)

			if newEc2.clusterTag != "" {
				c.osdResources[newEc2.clusterTag] = append(c.osdResources[newEc2.clusterTag], newEc2)
			}
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
		var hasRhmiTag bool
		newVpc.clusterTag, hasRhmiTag = extractClusterTag(newVpc.tags)
		if hasRhmiTag {
			c.rhmiResources[newVpc.clusterTag] = append(c.rhmiResources[newVpc.clusterTag], newVpc)
		} else {
			c.vpcs = append(c.vpcs, newVpc)
		}
	}
	return nil
}

func (c *cleanupAwsAccountCmd) cleanupUnusedVeleroS3Buckets(ctx context.Context) error {
	log.Debug("Deleting S3 Buckets")
	for _, b := range c.s3Buckets {
		if isVeleroBucket(b.ID) {
			if _, ok := c.osdResources[b.clusterTag]; !ok && b.clusterTag != "" {
				if c.dryRun {
					log.Infof("would delete a bucket '%s' (cluster tag '%s')\n", b.ID, b.clusterTag)
				} else {
					if err := c.removeS3Bucket(b.ID, ctx); err != nil {
						log.Warnf("Failed to delete S3 bucket '%s' (cluster tag '%s'). It might be already deleted\n", b.ID, b.clusterTag)
						log.Debug(err)
					} else {
						log.Infof("S3 bucket '%s' (cluster tag '%s') successfully deleted\n", b.ID, b.clusterTag)
						c.deletedResources = append(c.deletedResources, b)
					}
				}
			} else {
				c.osdResources[b.clusterTag] = append(c.osdResources[b.clusterTag], b)
			}
		}
	}
	return nil
}

func (c *cleanupAwsAccountCmd) cleanupVpcs() error {
	log.Debug("Deleting VPCs")
	for _, vpc := range c.vpcs {
		if _, ok := c.osdResources[vpc.clusterTag]; !ok && vpc.clusterTag != "" || len(vpc.tags) == 0 {
			if c.dryRun {
				log.Infof("would delete a vpc '%s' (cluster tag: '%s')\n", vpc.ID, vpc.clusterTag)
			} else {
				_, err := c.ec2.DeleteVpc(&ec2.DeleteVpcInput{
					VpcId: aws.String(vpc.ID),
				})
				if err != nil {
					log.Warnf("Failed to delete VPC '%s' (cluster tag '%s'). It might be deleted already or it still contains dependencies\n", vpc.ID, vpc.clusterTag)
					log.Debug(err)
				} else {
					log.Infof("VPC '%s' (cluster tag '%s') successfully deleted\n", vpc.ID, vpc.clusterTag)
					c.deletedResources = append(c.deletedResources, vpc)
				}
			}
		} else {
			c.osdResources[vpc.clusterTag] = append(c.osdResources[vpc.clusterTag], vpc)
		}
	}
	return nil
}

func (c *cleanupAwsAccountCmd) cleanupWithClusterService(tag string) error {

	log.Infof("About to clean up AWS resources for cluster tag '%s' with cluster-service", tag)

	err := wait.PollImmediate(30*time.Second, 20*time.Minute, func() (bool, error) {

		report, err := c.clusterService.DeleteResourcesForCluster(tag, map[string]string{}, c.dryRun)

		if err != nil {
			log.Debug("error when running clusterService", err)
			os.Exit(1)
		}

		for _, item := range report.Items {
			log.Debug(item.Name, item.ID, item.Action, item.ActionStatus)
		}

		if !report.AllItemsComplete() {
			log.Infof("AWS resources with cluster tag '%s' are still being deleted. Retrying in 30 seconds...", tag)
		}
		return report.AllItemsComplete(), nil

	})

	if err != nil {
		return err
	}
	log.Infof("Finished cleaning up AWS resources for cluster tag '%s'", tag)

	return nil
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

func isVeleroBucket(bucketName string) bool {
	return strings.Contains(bucketName, veleroS3BucketPrefix)
}

func extractClusterTag(tags map[string]string) (clusterTag string, hasRhmiTag bool) {
	for key, value := range tags {
		if strings.Contains(key, osdClusterTagPrefix) {
			return strings.Replace(key, osdClusterTagPrefix, "", 1), false
		}
		if strings.Contains(key, rhmiResourceTagPrefix) {
			return value, true
		}
		if strings.Contains(key, managedVeleroTagPrefix) {
			return value, false
		}
	}
	return
}

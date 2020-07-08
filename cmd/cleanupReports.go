package cmd

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

const (
	archiveFolderName = "archive"
)

type objectTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type cleanupConfig struct {
	Bucket string      `json:"bucket"`
	Tags   []objectTag `json:"tags"`
}

type cleanupConfigList struct {
	Configs []cleanupConfig `json:"configs"`
}

type cleanupReportsCmd struct {
	config    *cleanupConfigList
	s3        s3iface.S3API
	s3Deleter s3manageriface.BatchDelete
}

type cleanupReportsCmdFlags struct {
	configFile string
}

type cleanupResult struct {
	movedObjects []*s3.Object
}

func init() {
	f := &cleanupReportsCmdFlags{}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up the reports. Move the ones that have been processed to the 'archive/' sub directory in the bucket.",
		Run: func(cmd *cobra.Command, args []string) {
			awsKeyId, err := requireValue(AWSAccessKeyIDEnv)
			if err != nil {
				handleError(err)
			}
			awsSecretKey, err := requireValue(AWSSecretAccessKeyEnv)
			if err != nil {
				handleError(err)
			}
			sess := session.Must(session.NewSession(&aws.Config{
				Region:      aws.String(AWSDefaultRegion),
				Credentials: credentials.NewStaticCredentials(awsKeyId, awsSecretKey, ""),
			}))

			c, err := newCleanupReportsCmd(f, sess)
			if err != nil {
				handleError(err)
			}
			if err := c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}
	reportCmd.AddCommand(cmd)
	cmd.Flags().StringVar(&f.configFile, "config-file", "", "Path to the configuration file for this command")
	cmd.MarkFlagRequired("config-file")
}

func newCleanupReportsCmd(f *cleanupReportsCmdFlags, session *session.Session) (*cleanupReportsCmd, error) {
	c := &cleanupConfigList{}
	err := utils.PopulateObjectFromYAML(f.configFile, c)
	if err != nil {
		return nil, err
	}
	s3 := s3.New(session)
	deleter := s3manager.NewBatchDelete(session)
	return &cleanupReportsCmd{
		config:    c,
		s3:        s3,
		s3Deleter: deleter,
	}, nil
}

func (c *cleanupReportsCmd) run(ctx context.Context) error {
	tasks := make([]utils.Task, len(c.config.Configs))
	for i, conf := range c.config.Configs {
		config := conf
		t := func() (utils.TaskResult, error) {
			return c.cleanupObjectsForBucket(ctx, config)
		}
		tasks[i] = t
	}
	if _, err := utils.ParallelLimit(ctx, tasks, len(c.config.Configs)); err != nil {
		return err
	}
	fmt.Println("[All] Process completed")
	return nil
}

func (c *cleanupReportsCmd) cleanupObjectsForBucket(ctx context.Context, config cleanupConfig) (*cleanupResult, error) {
	bucket := config.Bucket
	fmt.Println(fmt.Sprintf("[%s] List objects in bucket", bucket))
	objects, err := c.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{Bucket: &bucket, Delimiter: aws.String("/")})
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("[%s] Found %d objects", bucket, len(objects.Contents)))
	toCopy := []*s3.Object{}
	for _, o := range objects.Contents {
		ok, err := c.shouldCleanup(ctx, bucket, o, config.Tags)
		if err != nil {
			fmt.Println(fmt.Sprintf("[%s] Skip object %s due to error: %v", bucket, *o.Key, err))
			continue
		}
		if ok {
			fmt.Println(fmt.Sprintf("[%s] Object %s has matched tags and will be moved", bucket, *o.Key))
			toCopy = append(toCopy, o)
		} else {
			fmt.Println(fmt.Sprintf("[%s] Skip object %s as it doesn't have the required tags", bucket, *o.Key))
		}
	}
	copied := c.copyObjects(ctx, bucket, archiveFolderName, toCopy)
	if err = c.deleteObjects(ctx, bucket, copied); err != nil {
		return nil, err
	}
	return &cleanupResult{movedObjects: copied}, nil
}

func (c *cleanupReportsCmd) shouldCleanup(ctx context.Context, bucket string, object *s3.Object, tags []objectTag) (bool, error) {
	fmt.Println(fmt.Sprintf("[%s] Listing tags for object %s", bucket, *object.Key))
	t, err := c.s3.GetObjectTaggingWithContext(ctx, &s3.GetObjectTaggingInput{
		Bucket: &bucket,
		Key:    object.Key,
	})
	if err != nil {
		return false, err
	}
	cleanup := true
	for _, tag := range tags {
		if !hasTag(t.TagSet, tag.Key, tag.Value) {
			cleanup = false
			break
		}
	}
	return cleanup, nil
}

func (c *cleanupReportsCmd) copyObjects(ctx context.Context, bucket string, toFolder string, objects []*s3.Object) []*s3.Object {
	copiedObjects := []*s3.Object{}
	if len(objects) == 0 {
		fmt.Println(fmt.Sprintf("[%s] No objects to copy", bucket))
		return copiedObjects
	}
	fmt.Println(fmt.Sprintf("[%s] Copying %d objects to %s/", bucket, len(objects), toFolder))
	for _, o := range objects {
		input := &s3.CopyObjectInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String(fmt.Sprintf("%s/%s", toFolder, *o.Key)),
			CopySource: aws.String(fmt.Sprintf("%s/%s", bucket, *o.Key)),
		}
		_, err := c.s3.CopyObjectWithContext(ctx, input)
		if err != nil {
			fmt.Println(fmt.Sprintf("[%s] Failed to copy object %s due to error: %v", bucket, *o.Key, err))
		} else {
			fmt.Println(fmt.Sprintf("[%s] Object %s copied to %s/%s", bucket, *o.Key, toFolder, *o.Key))
			copiedObjects = append(copiedObjects, o)
		}
	}
	fmt.Println(fmt.Sprintf("[%s] Copied %d objects to %s", bucket, len(copiedObjects), toFolder))
	return copiedObjects
}

func (c *cleanupReportsCmd) deleteObjects(ctx context.Context, bucket string, toDelete []*s3.Object) error {
	if len(toDelete) == 0 {
		fmt.Println(fmt.Sprintf("[%s] No objects to delete", bucket))
		return nil
	}
	fmt.Println(fmt.Sprintf("[%s] Deleting %d objects", bucket, len(toDelete)))
	batch := []s3manager.BatchDeleteObject{}
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
	fmt.Println(fmt.Sprintf("[%s] %d objects deleted", bucket, len(toDelete)))
	return nil
}

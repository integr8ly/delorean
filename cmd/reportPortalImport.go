package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/reportportal"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const (
	reportPortalTagKey = "rp"
	reportPortalTagVal = "true"

	defaultImportWorkers = 10

	metadataFileName = "metadata.json"
	rpTokenKey       = "report_portal_token"
)

type reportPortalImportCmdFlags struct {
	bucket              string
	reportPortalProject string
	noTagging           bool
}

type reportPortalImportCmd struct {
	fromBucket      string
	rpLaunchService reportportal.RPLaunchManager
	s3              s3iface.S3API
	s3downloader    s3manageriface.DownloaderAPI
	rpProjectName   string
	noTagging       bool
}

type reportProcessResult struct {
	s3ObjKey   string
	rpLaunchId string
}

type testMetadata struct {
	Name        string `json:"name"`
	RHMIVersion string `json:"rhmiVersion"`
	JobURL      string `json:"jobURL"`
}

func init() {
	f := &reportPortalImportCmdFlags{}
	cmd := &cobra.Command{
		Use:   "reportportal-import",
		Short: "Import test results from s3 to ReportPortal",
		Run: func(cmd *cobra.Command, args []string) {
			awsKeyId, err := requireValue(AWSAccessKeyIDEnv)
			if err != nil {
				handleError(err)
			}
			awsSecretKey, err := requireValue(AWSSerectAccessKeyEnv)
			if err != nil {
				handleError(err)
			}
			rpToken, err := requireValue(rpTokenKey)
			if err != nil {
				handleError(err)
			}

			sess := session.Must(session.NewSession(&aws.Config{
				Region:      aws.String(AWSDefaultRegion),
				Credentials: credentials.NewStaticCredentials(awsKeyId, awsSecretKey, ""),
			}))

			c, err := newReportPortalImportCmd(f, sess, rpToken)
			if err != nil {
				handleError(err)
			}
			if err := c.run(cmd.Context()); err != nil {
				handleError(err)
			}
		},
	}
	reportCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.bucket, "bucket", "b", "", "The S3 bucket to download reports from")
	cmd.MarkFlagRequired("bucket")
	cmd.Flags().StringVarP(&f.reportPortalProject, "project", "p", "", "The name of the ReportPortal project")
	cmd.MarkFlagRequired("project")

	cmd.Flags().String("rp-token", "", fmt.Sprintf("The token (UUID) to access the ReportPortal API. Can be set via the %s env var", strings.ToUpper(rpTokenKey)))
	viper.BindPFlag(rpTokenKey, cmd.Flags().Lookup("rp-token"))

	cmd.Flags().BoolVar(&f.noTagging, "no-tagging", false, "Do not add new tags to the AWS resources, for testing purposes.")
}

func newRPClient(token string) *reportportal.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return reportportal.NewClient(tc)
}

func newReportPortalImportCmd(f *reportPortalImportCmdFlags, session *session.Session, rpToken string) (*reportPortalImportCmd, error) {
	rp := newRPClient(rpToken)
	s3 := s3.New(session)
	s3Downloader := s3manager.NewDownloader(session)
	return &reportPortalImportCmd{
		fromBucket:      f.bucket,
		rpLaunchService: rp.Launches,
		s3:              s3,
		s3downloader:    s3Downloader,
		rpProjectName:   f.reportPortalProject,
		noTagging:       f.noTagging,
	}, nil
}

func (c *reportPortalImportCmd) run(ctx context.Context) error {
	fmt.Println(fmt.Sprintf("[All] Listing objects from bucket %s", c.fromBucket))
	o, err := c.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{Bucket: &c.fromBucket})
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("[All] Found %d objects to process", len(o.Contents)))
	tasks := make([]utils.Task, len(o.Contents))
	for i, obj := range o.Contents {
		f := obj
		t := func() (utils.TaskResult, error) {
			return c.processReportFile(ctx, f)
		}
		tasks[i] = t
	}

	if _, err := utils.ParallelLimit(ctx, tasks, defaultImportWorkers); err != nil {
		return err
	}
	fmt.Println("[All] Process completed")
	return nil
}

func (c *reportPortalImportCmd) processReportFile(ctx context.Context, object *s3.Object) (*reportProcessResult, error) {
	fmt.Println(fmt.Sprintf("[%s] Start processing object", *object.Key))
	tags, err := c.s3.GetObjectTaggingWithContext(ctx, &s3.GetObjectTaggingInput{
		Bucket: &c.fromBucket,
		Key:    object.Key,
	})
	if err != nil {
		return nil, err
	}
	if hasTag(tags.TagSet, reportPortalTagKey, reportPortalTagVal) {
		fmt.Println(fmt.Sprintf("[%s] File in bucket %s has been processed already. Ignored.", *object.Key, c.fromBucket))
		return &reportProcessResult{}, nil
	}
	fmt.Println(fmt.Sprintf("[%s] Downloading file from s3 bucket %s", *object.Key, c.fromBucket))
	// download object to a tmp dir
	downloaded, err := utils.DownloadS3ObjectToTempDir(ctx, c.s3downloader, c.fromBucket, *object.Key)
	if err != nil {
		return nil, err
	}
	defer os.Remove(downloaded)

	fmt.Println(fmt.Sprintf("[%s] File Downloaded. Extracting metadata.json file.", *object.Key))
	// get the metadata.json file
	b, err := utils.ReadFileFromZip(downloaded, metadataFileName)
	if err != nil {
		return nil, err
	}
	m := &testMetadata{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("[%s] Metadata.json file is loaded. Uploading results to ReportPortal: %s", *object.Key, reportportal.BaseURL))

	// upload it to ReportPortal
	importResp, err := c.rpLaunchService.Import(ctx, c.rpProjectName, downloaded, m.Name)
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("[%s] File uploaded. Update the launch to add tags with id %s", *object.Key, importResp.GetLaunchId()))
	// update the launch obj to add a bit more info
	update := &reportportal.RPLaunchUpdateInput{
		Description: m.JobURL,
		Tags:        []string{m.Name, m.RHMIVersion},
	}
	if _, err := c.rpLaunchService.Update(ctx, c.rpProjectName, importResp.GetLaunchId(), update); err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("[%s] Launch updated. Id = %s", *object.Key, importResp.GetLaunchId()))
	if !c.noTagging {
		// update the tags on the obj
		fmt.Println(fmt.Sprintf("[%s] Adding tag %s=%s to s3 object", *object.Key, reportPortalTagKey, reportPortalTagVal))
		t := append(tags.TagSet, &s3.Tag{
			Key:   aws.String(reportPortalTagKey),
			Value: aws.String(reportPortalTagVal),
		})
		if _, err = c.s3.PutObjectTaggingWithContext(ctx, &s3.PutObjectTaggingInput{
			Bucket:  aws.String(c.fromBucket),
			Key:     object.Key,
			Tagging: &s3.Tagging{TagSet: t},
		}); err != nil {
			return nil, err
		}
		fmt.Println(fmt.Sprintf("[%s] Tags updated", *object.Key))
	} else {
		fmt.Println(fmt.Sprintf("[%s] Skip adding tags", *object.Key))
	}

	return &reportProcessResult{
		s3ObjKey:   *object.Key,
		rpLaunchId: importResp.GetLaunchId(),
	}, nil
}

func hasTag(tags []*s3.Tag, key string, val string) bool {
	for _, t := range tags {
		if *t.Key == key && *t.Value == val {
			return true
		}
	}
	return false
}

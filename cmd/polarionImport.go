package cmd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/polarion"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/jstemmer/go-junit-report/formatter"
	"github.com/spf13/cobra"
)

const (
	polarionTagKey = "polarion"
	polarionTagVal = "true"

	integreatlyOperatorJUnit = "integreatly-operator-test/results/junit-integreatly-operator.xml"

	polarionImportURL        = "https://polarion.engineering.redhat.com/polarion/import"
	polarionImportStagingURL = "https://polarion.stage.engineering.redhat.com/polarion/import"
)

type polarionImportCmdFlags struct {
	bucket string
	stage  bool
}

type polarionImportCmd struct {
	fromBucket       string
	s3               s3iface.S3API
	s3downloader     s3manageriface.DownloaderAPI
	polarionURL      string
	polarionImporter polarion.XUnitImporterService
}

func init() {
	f := &polarionImportCmdFlags{}
	cmd := &cobra.Command{
		Use:   "polarion-import",
		Short: "Import test results from s3 to Polarion",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newPolarionImportCmd(f)
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

	cmd.Flags().BoolVar(&f.stage, "stage", false, "Create the release in the Polarion staging environment")
}

func newPolarionImportCmd(f *polarionImportCmdFlags) (*polarionImportCmd, error) {

	awsAccessKeyID, err := requireValue(AWSAccessKeyIDEnv)
	if err != nil {
		return nil, err
	}
	awsSecretAccessKey, err := requireValue(AWSSecretAccessKeyEnv)
	if err != nil {
		return nil, err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(AWSDefaultRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})

	s3 := s3.New(session)
	s3Downloader := s3manager.NewDownloader(session)

	polarionUsername, err := requireValue(PolarionUsernameKey)
	if err != nil {
		return nil, err
	}

	polarionPassword, err := requireValue(PolarionPasswordKey)
	if err != nil {
		return nil, err
	}

	var url string
	if f.stage {
		url = polarionImportStagingURL
	} else {
		url = polarionImportURL
	}

	importer := polarion.NewXUnitImporter(url, polarionUsername, polarionPassword)

	return &polarionImportCmd{
		fromBucket:       f.bucket,
		s3:               s3,
		s3downloader:     s3Downloader,
		polarionURL:      url,
		polarionImporter: importer,
	}, nil
}

func (c *polarionImportCmd) run(ctx context.Context) error {
	o, err := c.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{Bucket: &c.fromBucket, Delimiter: aws.String("/")})
	if err != nil {
		return err
	}

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

	return nil
}

func (c *polarionImportCmd) processReportFile(ctx context.Context, object *s3.Object) (interface{}, error) {
	tags, err := c.s3.GetObjectTaggingWithContext(ctx, &s3.GetObjectTaggingInput{
		Bucket: &c.fromBucket,
		Key:    object.Key,
	})
	if err != nil {
		return nil, err
	}
	if hasTag(tags.TagSet, polarionTagKey, polarionTagVal) {
		fmt.Println(fmt.Sprintf("[%s] file in bucket %s has been processed already. Ignored.", *object.Key, c.fromBucket))
		return &reportProcessResult{}, nil
	}

	if !strings.HasSuffix(*object.Key, ".zip") {
		fmt.Println(fmt.Sprintf("[%s] file in bucket %s is ignored as it is not a zip file", *object.Key, c.fromBucket))
		return &reportProcessResult{}, nil
	}

	// download object to a tmp dir
	fmt.Println(fmt.Sprintf("[%s] downloading file from s3 bucket %s", *object.Key, c.fromBucket))
	downloaded, err := utils.DownloadS3ObjectToTempDir(ctx, c.s3downloader, c.fromBucket, *object.Key)
	if err != nil {
		return nil, err
	}
	defer os.Remove(downloaded)

	// get the metadata.json file
	b, err := utils.ReadFileFromZip(downloaded, metadataFileName)
	if err != nil {
		return nil, err
	}
	m := &testMetadata{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, err
	}

	// upload it to Polarion
	fmt.Println(fmt.Sprintf("[%s] uploading results to Polarion", *object.Key))
	err = c.importToPolarion(*object.Key, m, downloaded)
	if err != nil {
		return nil, err
	}

	// update the tags on the obj
	t := append(tags.TagSet, &s3.Tag{
		Key:   aws.String(polarionTagKey),
		Value: aws.String(polarionTagVal),
	})
	if _, err = c.s3.PutObjectTaggingWithContext(ctx, &s3.PutObjectTaggingInput{
		Bucket:  aws.String(c.fromBucket),
		Key:     object.Key,
		Tagging: &s3.Tagging{TagSet: t},
	}); err != nil {
		return nil, err
	}

	return &struct{}{}, nil
}

func (c *polarionImportCmd) importToPolarion(key string, metadata *testMetadata, zipfile string) error {

	// Do not import master/nightly tests
	if metadata.RHMIVersion == "" {
		fmt.Printf("[%s] ignore test results %s %s\n", key, metadata.Name, metadata.RHMIVersion)
		return nil
	}

	version, err := utils.NewRHMIVersion(metadata.RHMIVersion)
	if err != nil {
		return err
	}

	// Get integreatly-operator JUnit
	b, err := utils.ReadFileFromZip(zipfile, integreatlyOperatorJUnit)
	if err != nil {
		return err
	}
	junit := &formatter.JUnitTestSuites{}
	if err := xml.Unmarshal(b, junit); err != nil {
		return err
	}

	title := fmt.Sprintf("RHMI %s %s Automated Tests", version.String(), metadata.Name)
	xunit, err := polarion.JUnitToPolarionXUnit(junit, polarionProjectID, title, version.PolarionMilestoneId())
	if err != nil {
		return err
	}

	jobID, err := c.polarionImporter.Import(xunit)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] polarion job started with id %d\n", key, jobID)
	fmt.Printf("[%s] logs: %s/xunit-log?jobId=%d\n", key, c.polarionURL, jobID)

	for {
		time.Sleep(2 * time.Second)

		status, err := c.polarionImporter.GetJobStatus(jobID)
		if err != nil {
			panic(err)
		}

		exit := false
		switch status {
		case polarion.ReadyStatus:
		case polarion.RunningStatus:
			fmt.Printf("[%s] polarion job is %s\n", key, status)
		case polarion.SuccessStatus:
			fmt.Printf("[%s] polarion job completed successfully\n", key)
			exit = true
		default:
			return fmt.Errorf("[%s] unknown job status %s", key, status)
		}

		if exit {
			break
		}
	}

	return nil
}

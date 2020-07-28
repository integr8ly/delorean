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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

const (
	datahubTagKey = "datahub"
	datahubTagVal = "true"

	downtimeReportFilename = "downtime-report.yaml"
	pushgateway            = "http://pushgateway-dh-prod-monitoring.cloud.datahub.psi.redhat.com:9091"
	jobName                = "rhmi-product-downtime"
)

type datahubImportCmdFlags struct {
	bucket      string
	reportName  string
	pushgateway string
	jobName     string
}

type datahubImportCmd struct {
	fromBucket   string
	s3           s3iface.S3API
	s3Downloader s3manageriface.DownloaderAPI
	reportName   string
	pushgateway  string
	jobName      string
}

var downtimeCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "rhmi_product_downtime",
	Help: "Downtime count in seconds",
})

func init() {
	f := &datahubImportCmdFlags{}
	cmd := &cobra.Command{
		Use:   "datahub-import",
		Short: "Import test results from s3 to DataHub",
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

			c, err := newDatahubImportCmd(f, sess)
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
	cmd.Flags().StringVarP(&f.reportName, "reportname", "r", "", "The filename of the report to process")
	cmd.Flags().StringVarP(&f.pushgateway, "pushgateway", "p", "", "The url of the prometheus pushgateway")
	cmd.Flags().StringVarP(&f.jobName, "jobname", "j", "", "The jobname for the prometheus metrics")
}

func newDatahubImportCmd(f *datahubImportCmdFlags, session *session.Session) (*datahubImportCmd, error) {
	s3i := s3.New(session)
	s3Downloader := s3manager.NewDownloader(session)

	return &datahubImportCmd{
		fromBucket:   f.bucket,
		s3Downloader: s3Downloader,
		s3:           s3i,
		reportName:   f.reportName,
		pushgateway:  f.pushgateway,
		jobName:      f.jobName,
	}, nil
}

func (c *datahubImportCmd) run(ctx context.Context) error {
	fmt.Println(fmt.Sprintf("[All] Listing objects from bucket %s", c.fromBucket))
	o, err := c.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{Bucket: &c.fromBucket, Delimiter: aws.String("/")})
	if err != nil {
		return err
	}

	tasks := make([]utils.Task, len(o.Contents))

	if c.reportName == "" {
		c.reportName = downtimeReportFilename
	}
	if c.pushgateway == "" {
		c.pushgateway = pushgateway
	}
	if c.jobName == "" {
		c.jobName = jobName
	}

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

func (c *datahubImportCmd) processReportFile(ctx context.Context, object *s3.Object) (interface{}, error) {
	if !strings.HasPrefix(*object.Key, "downtime-report") {
		fmt.Println(fmt.Sprintf("[%s] Skipping processing object", *object.Key))
		return &struct{}{}, nil
	}

	fmt.Println(fmt.Sprintf("[%s] Start processing object", *object.Key))

	tags, err := c.s3.GetObjectTaggingWithContext(ctx, &s3.GetObjectTaggingInput{
		Bucket: &c.fromBucket,
		Key:    object.Key,
	})
	if err != nil {
		return nil, err
	}

	if hasTag(tags.TagSet, datahubTagKey, datahubTagVal){
		fmt.Println(fmt.Sprintf("[%s] file in bucket %s has been processed already. Ignored.", *object.Key, c.fromBucket))
		return nil, nil
	}

	fmt.Println(fmt.Sprintf("[%s] Downloading file from s3 bucket %s", *object.Key, c.fromBucket))
	// download object to a tmp dir
	downloaded, err := utils.DownloadS3ObjectToTempDir(ctx, c.s3Downloader, c.fromBucket, *object.Key)
	if err != nil {
		return nil, err
	}
	defer os.Remove(downloaded)

	qr := &queryResults{}
	err = utils.PopulateObjectFromYAML(downloaded, qr)
	if err != nil {
		return nil, err
	}

	fmt.Println(fmt.Sprintf("[%s] Downtime Report file is loaded. Uploading to prometheus at %s", *object.Key, c.pushgateway))

	// Get version string
	ver, err := utils.NewRHMIVersion(qr.Version)
	if err != nil {
		return nil, err
	}
	var name string
	for _, i := range qr.Results {
		name = strings.Split(i.Name,"_")[0]
		if len(i.v.String()) > 0 {
			// The metric value comes with the query so split it to get the int value we care about
			count, err := parseValue(i.v.String())
			if err != nil {
				return 0, err
			}
			downtimeCount.Set(float64(count))

		}
		// push the metric
		pusher := push.New(c.pushgateway, c.jobName)
		pusher.Collector(downtimeCount).Grouping("product", name).Grouping("query", i.Name).Grouping("version", ver.String())
		err = pusher.Push()

		if err != nil {
			e := fmt.Errorf("failed to push to %s: %s", c.pushgateway, err)
			return nil, e
		}
	}
	// update tags
	t := append(tags.TagSet, &s3.Tag{
		Key:   aws.String(datahubTagKey),
		Value: aws.String(datahubTagVal),
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

func parseValue(in string) (int, error) {
	p := strings.Split(in, "=>")
	p = strings.Split(p[1], "@")
	r := strings.TrimSpace(p[0])
	n, err := strconv.Atoi(r)
	if err != nil {
		return 0, err
	}
	return n, nil
}

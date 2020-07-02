package cmd

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"time"
)

type MetaData struct {
	Name    string `json:"name"`
	Version string `json:"rhmiVersion"`
	Job     string `json:"jobURL"`
}

type exportCmdFlags struct {
	testResultsDir string
	pipelineName   string
	rhmiVersion    string
	jobLink        string
	s3bucket       string
}

type exportCmd struct {
	metadataFile string
	metadataObj  MetaData
	zippedDir    string
	zipFile      string
	bucket       string
	uploader     s3manageriface.UploaderAPI
}

func init() {
	f := &exportCmdFlags{}
	cmd := &cobra.Command{
		Use:   "export-results",
		Short: "Export RHMI tests and products tests results to s3 bucket",
		Run: func(cmd *cobra.Command, args []string) {

			awsKeyId, err := requireValue(AWSAccessKeyIDEnv)
			if err != nil {
				handleError(err)
			}
			awsSecretKey, err := requireValue(AWSSecretAccessKeyEnv)
			if err != nil {
				handleError(err)
			}

			s := session.Must(session.NewSession(&aws.Config{
				Region:      aws.String(AWSDefaultRegion),
				Credentials: credentials.NewStaticCredentials(awsKeyId, awsSecretKey, ""),
			}))

			c, err := newExportCmd(f, s)
			if err != nil {
				handleError(err)
			}
			err = c.run(cmd.Context())
			if err != nil {
				handleError(err)
			}
		},
	}

	reportCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.testResultsDir, "results-dir", "d", "", "Absolute path of the test results directory to be exported")
	cmd.MarkFlagRequired("results-dir")
	cmd.Flags().StringVarP(&f.pipelineName, "pipeline-name", "n", "", "Name of the pipeline that created results")
	cmd.MarkFlagRequired("pipeline-name")
	cmd.Flags().StringVarP(&f.jobLink, "job-link", "j", "", "Link to the job that created results")
	cmd.MarkFlagRequired("job-link")
	cmd.Flags().StringVarP(&f.rhmiVersion, "rhmi-version", "v", "", "RHMI version of results")
	cmd.MarkFlagRequired("rhmi-version")
	cmd.Flags().StringVarP(&f.s3bucket, "s3-bucket", "b", "", "AWS s3 bucket name")
	cmd.MarkFlagRequired("s3-bucket")
}

func newExportCmd(f *exportCmdFlags, session *session.Session) (*exportCmd, error) {
	metadataFile := f.testResultsDir + metadataFileName
	metadataObj := MetaData{
		Name:    f.pipelineName,
		Version: f.rhmiVersion,
		Job:     f.jobLink,
	}
	zipFile := f.pipelineName + "-" +
		time.Now().Format("2006-01-02-03-04-05") +
		".zip"
	uploader := s3manager.NewUploader(session)
	return &exportCmd{
		metadataFile: metadataFile,
		metadataObj:  metadataObj,
		zippedDir:    f.testResultsDir,
		zipFile:      zipFile,
		bucket:       f.s3bucket,
		uploader:     uploader,
	}, nil
}

func (c *exportCmd) run(ctx context.Context) error {

	// generate metadata.json inside results dir
	fmt.Println("Generating... " + c.metadataFile)
	err := utils.WriteObjectToJSON(c.metadataObj, c.metadataFile)
	if err != nil {
		return err
	}

	// zip the results
	fmt.Println("Generating... " + c.zippedDir + c.zipFile)
	err = utils.ZipFolder(c.zippedDir, c.zippedDir+c.zipFile)
	if err != nil {
		return err
	}

	// export the zip
	fmt.Println("Exporting... " + c.zipFile)
	location, err := utils.UploadFileToS3(ctx, c.uploader, c.bucket, c.zippedDir, c.zipFile)
	if err != nil {
		return err
	}
	fmt.Println("Exported: " + location)
	return nil
}

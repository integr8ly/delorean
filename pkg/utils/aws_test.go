package utils

import (
	"context"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestDownloadS3ObjectToTempDir(t *testing.T) {
	f := "./testdata/yaml/sample-one.yaml"
	cases := []struct {
		description string
		testFile    string
		downloader  s3manageriface.DownloaderAPI
	}{
		{
			description: "success",
			downloader: &MockS3Downloader{
				DownloadFunc: func(w io.WriterAt, input *s3.GetObjectInput) (i int64, err error) {
					c, err := ioutil.ReadFile(f)
					if err != nil {
						return 0, err
					}
					b, err := w.WriteAt(c, 0)
					if err != nil {
						return 0, err
					}
					return int64(b), err
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			p, err := DownloadS3ObjectToTempDir(context.TODO(), c.downloader, "test", "test/sample.yaml")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer os.Remove(p)
			_, err = os.Stat(p)
			if err != nil {
				t.Fatalf("error getting the info for file: %s, err: %v", p, err)
			}
			wanted, _ := ioutil.ReadFile(f)
			actual, err := ioutil.ReadFile(p)
			if string(wanted) != string(actual) {
				t.Fatalf("file content doesn't match. wanted: %s, actual: %s", wanted, actual)
			}
		})
	}
}

package utils

import (
	"context"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// Download an s3 object with the given key in the given bucket to a temp directory, and return the path to the temp file.
// The key will be used as the file name, and all "/" in the key will be replaced by "_" if there's any.
func DownloadS3ObjectToTempDir(ctx context.Context, downloader s3manageriface.DownloaderAPI, bucket string, key string) (string, error) {
	dir, err := ioutil.TempDir("", "delorean-s3-")
	if err != nil {
		return "", nil
	}
	o := path.Join(dir, strings.ReplaceAll(key, "/", "_"))
	outFile, err := os.Create(o)
	if err != nil {
		return "", nil
	}
	defer outFile.Close()
	if err != nil {
		return "", err
	}
	i := &s3.GetObjectInput{Key: &key, Bucket: &bucket}
	if _, err = downloader.DownloadWithContext(ctx, outFile, i); err != nil {
		return "", err
	}
	return o, nil
}

package utils

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"io/ioutil"
	"net/http"
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

func UploadFileToS3(ctx context.Context, uploader s3manageriface.UploaderAPI, bucket string, fileDir string, fileName string) (string, error) {

	// Open the file for use
	file, err := os.Open(fileDir + fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	buffer := make([]byte, fileInfo.Size())
	file.Read(buffer)

	i := &s3manager.UploadInput{
		Body:        bytes.NewReader(buffer),
		Bucket:      aws.String(bucket),
		ContentType: aws.String(http.DetectContentType(buffer)),
		Key:         aws.String(fileName),
	}
	output, err := uploader.UploadWithContext(ctx, i)
	return output.Location, err
}

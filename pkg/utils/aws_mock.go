package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"io"
)

type MockS3Downloader struct {
	s3manageriface.DownloaderAPI
	DownloadFunc func(io.WriterAt, *s3.GetObjectInput) (int64, error)
}

func (m *MockS3Downloader) DownloadWithContext(_ aws.Context, o io.WriterAt, input *s3.GetObjectInput, _ ...func(*s3manager.Downloader)) (int64, error) {
	return m.DownloadFunc(o, input)
}

type MockS3Uploader struct {
	s3manageriface.UploaderAPI
	UploadFunc func(*s3manager.UploadInput) (*s3manager.UploadOutput, error)
}

func (m *MockS3Uploader) UploadWithContext(_ aws.Context, input *s3manager.UploadInput, _ ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return m.UploadFunc(input)
}

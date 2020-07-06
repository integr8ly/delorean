package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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

type MockS3API struct {
	s3iface.S3API
	ListObjsFunc      func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	GetObjTaggingFunc func(*s3.GetObjectTaggingInput) (*s3.GetObjectTaggingOutput, error)
	PutObjTaggingFunc func(*s3.PutObjectTaggingInput) (*s3.PutObjectTaggingOutput, error)
	CopyObjectFunc    func(*s3.CopyObjectInput) (*s3.CopyObjectOutput, error)
}

func (m *MockS3API) ListObjectsV2WithContext(_ aws.Context, input *s3.ListObjectsV2Input, _ ...request.Option) (*s3.ListObjectsV2Output, error) {
	return m.ListObjsFunc(input)
}

func (m *MockS3API) GetObjectTaggingWithContext(_ aws.Context, input *s3.GetObjectTaggingInput, _ ...request.Option) (*s3.GetObjectTaggingOutput, error) {
	return m.GetObjTaggingFunc(input)
}

func (m *MockS3API) PutObjectTaggingWithContext(_ aws.Context, input *s3.PutObjectTaggingInput, _ ...request.Option) (*s3.PutObjectTaggingOutput, error) {
	return m.PutObjTaggingFunc(input)
}

func (m *MockS3API) CopyObjectWithContext(_ aws.Context, input *s3.CopyObjectInput, _ ...request.Option) (*s3.CopyObjectOutput, error) {
	return m.CopyObjectFunc(input)
}

type MockS3BatchDeleter struct {
	s3manageriface.BatchDelete
	BatchDeleteFunc func(s3manager.BatchDeleteIterator) error
}

func (m *MockS3BatchDeleter) Delete(_ aws.Context, input s3manager.BatchDeleteIterator) error {
	return m.BatchDeleteFunc(input)
}

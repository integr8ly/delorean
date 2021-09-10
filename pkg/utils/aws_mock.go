package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
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
	//GetBucketLocationOutput *s3.GetBucketLocationOutput
	GetBucketTaggingFunc func(*s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error)
	ListBucketsOutput    *s3.ListBucketsOutput
	ListObjsFunc         func(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	GetObjTaggingFunc    func(*s3.GetObjectTaggingInput) (*s3.GetObjectTaggingOutput, error)
	PutObjTaggingFunc    func(*s3.PutObjectTaggingInput) (*s3.PutObjectTaggingOutput, error)
	CopyObjectFunc       func(*s3.CopyObjectInput) (*s3.CopyObjectOutput, error)
}

func (m *MockS3API) DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return nil, nil
}

func (m *MockS3API) GetBucketLocation(_ *s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return &s3.GetBucketLocationOutput{LocationConstraint: aws.String("us-east-1")}, nil
}

func (m *MockS3API) GetBucketTagging(input *s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	return m.GetBucketTaggingFunc(input)
}

func (m *MockS3API) ListBucketsWithContext(_ aws.Context, _ *s3.ListBucketsInput, _ ...request.Option) (*s3.ListBucketsOutput, error) {
	return m.ListBucketsOutput, nil
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

type MockEC2API struct {
	ec2iface.EC2API
	DescribeInstancesOutput *ec2.DescribeInstancesOutput
	DescribeVpcsOutput      *ec2.DescribeVpcsOutput
}

func (m *MockEC2API) DescribeInstancesWithContext(_ aws.Context, _ *ec2.DescribeInstancesInput, _ ...request.Option) (*ec2.DescribeInstancesOutput, error) {
	return m.DescribeInstancesOutput, nil
}

func (m *MockEC2API) DescribeVpcsWithContext(_ aws.Context, _ *ec2.DescribeVpcsInput, _ ...request.Option) (*ec2.DescribeVpcsOutput, error) {
	return m.DescribeVpcsOutput, nil
}

func (m *MockEC2API) DeleteVpc(_ *ec2.DeleteVpcInput) (*ec2.DeleteVpcOutput, error) {
	return nil, nil
}

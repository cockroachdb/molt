package datablobstorage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/mock"
)

type s3UploaderMock struct {
	s3manageriface.UploaderAPI
	mock.Mock
}

const AWSUploadFileMockErrMsg = "mocked error for uploading file for aws"

func (s *s3UploaderMock) Upload(
	input *s3manager.UploadInput, f ...func(*s3manager.Uploader),
) (*s3manager.UploadOutput, error) {
	return nil, errors.Newf(AWSUploadFileMockErrMsg)
}

func (s *s3UploaderMock) UploadWithContext(
	context aws.Context, input *s3manager.UploadInput, f ...func(*s3manager.Uploader),
) (*s3manager.UploadOutput, error) {
	return nil, errors.Newf(AWSUploadFileMockErrMsg)
}

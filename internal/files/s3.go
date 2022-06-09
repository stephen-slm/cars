package files

import (
	"bytes"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

type S3Files struct {
	config *S3Config
	s3     *s3.S3
}

func newS3Files(s3Config *S3Config) (S3Files, error) {
	s3Files := S3Files{config: s3Config}

	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return s3Files, err
	}

	s3Files.s3 = s3.New(sess)
	return s3Files, nil
}

func (s S3Files) WriteFile(id string, name string, data []byte) error {
	_, writeFileErr := s.s3.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(filepath.Join(id, name)),
	})

	if writeFileErr != nil {
		return errors.Wrapf(writeFileErr, "failed to create %s file", name)
	}

	return nil
}

func (s S3Files) GetFile(id string, name string) ([]byte, error) {
	output, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(filepath.Join(id, name)),
	})

	if err != nil {
		// nolint:errorlint // aws does not expose the error type
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == s3.ErrCodeNoSuchKey {
			return nil, errors.Wrapf(err, "cannot locate file %s", name)
		}

		return nil, errors.Wrapf(err, "failed to get the local file %s by id %s", name, id)
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(output.Body)

	return buffer.Bytes(), err
}

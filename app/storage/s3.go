package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3 struct {
	bucketName string
	uploader   *s3manager.Uploader
}

func NewS3(bucketName string) *S3 {
	sess := session.Must(session.NewSession())

	return &S3{
		bucketName: bucketName,
		uploader:   s3manager.NewUploader(sess),
	}
}

func (s3 *S3) FileExists(fileName string) (bool, error) {
	_, err := s3.uploader.S3.HeadObject(
		&awss3.HeadObjectInput{
			Bucket: &s3.bucketName,
			Key:    &fileName,
		},
	)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s3 *S3) UploadFile(fileName string, reader io.Reader) (string, error) {
	output, err := s3.uploader.UploadWithContext(
		context.TODO(),
		&s3manager.UploadInput{
			Bucket:       &s3.bucketName,
			Key:          &fileName,
			Body:         reader,
			CacheControl: aws.String("public, max-age=86400"), // 86400 = 24h = 24*60*60
			ACL:          aws.String("public-read"),
			// TODO: ContentType:  "",
		},
	)

	if err != nil {
		return "", err
	}

	return output.Location, nil
}

func (s3 *S3) BuildLink(fileName string) string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s3.bucketName, fileName)
}

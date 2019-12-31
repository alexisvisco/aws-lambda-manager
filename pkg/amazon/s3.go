package amazon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func S3BucketExist(sess *session.Session, bucketName string) bool {
	s := s3.New(sess)
	_, err := s.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	return err == nil
}

func S3CreateBucket(sess *session.Session, bucketName string) error {
	s := s3.New(sess)

	_, err := s.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func S3DeleteBucket(sess *session.Session, bucketName string) error {
	s := s3.New(sess)

	iter := s3manager.NewDeleteListIterator(s, &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})

	if err := s3manager.NewBatchDeleteWithClient(s).Delete(aws.BackgroundContext(), iter); err != nil {
		return err
	}

	_, err := s.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func S3ListObjects(sess *session.Session, bucketName string) (*s3.ListObjectsOutput, error) {
	s := s3.New(sess)
	return s.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})
}

func S3FileExist(sess *session.Session, bucketName string, sum string) bool {
	s := s3.New(sess)
	output, err := s.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return false
	}
	for _, content := range output.Contents {
		filename := *content.Key
		extension := filepath.Ext(filename)
		name := filename[0 : len(filename)-len(extension)]
		n := strings.SplitN(name, "-", 2)
		if len(n) == 2 {
			if n[1] == sum {
				return true
			}
		}
	}
	return false
}

func S3UploadFile(sess *session.Session, bucketName, sum, file string) (string, *s3manager.UploadOutput, error) {
	name := fmt.Sprintf("%d-%s.zip", time.Now().Unix(), sum)
	uploader := s3manager.NewUploader(sess)
	f, err := os.Open(file)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	output, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(name),
		Body:   f,
	})
	return name, output, err
}

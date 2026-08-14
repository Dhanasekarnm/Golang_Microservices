package s3storage

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func S3Connect() *session.Session {

	s3_endpoint := os.Getenv("S3_CONNECT")
	s3_acceskeyid := os.Getenv("S3_ACCESSKEYID")
	s3_secretacceskeyid := os.Getenv("S3_SECRETACCESSKEYID")
	s3_region := os.Getenv("S3_REGION")

	s3_protocol := strings.Split(s3_endpoint, "://")

	if len(s3_protocol) != 2 {
		log.Fatal("Wrong format of S3_Connect env variable")
	}
	if s3_protocol[0] != "http" && s3_protocol[0] != "https" {
		log.Fatal("Wrong format of S3_Connect  env variable")
	}

	disable_ssl := true
	if s3_protocol[0] == "https" {
		disable_ssl = false
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(s3_acceskeyid, s3_secretacceskeyid, ""),
		Endpoint:         aws.String(s3_endpoint),
		Region:           aws.String(s3_region),
		DisableSSL:       aws.Bool(disable_ssl),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession, err := session.NewSession(s3Config)

	if err != nil {
		log.Fatal(err)
	}
	return newSession
}

func S3DownloadCurrentVersion(session *session.Session, file *os.File, bucket *string, key *string) error {
	downloader := s3manager.NewDownloader(session)
	_, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	return err
}

func S3UploadFile(session *session.Session, file_path string, bucket string, key string) (string, string, error) {

	upFile, err := os.Open(file_path)
	if err != nil {
		log.Println("Unable to open the file", err)
		return "", "", err
	}
	defer upFile.Close()

	upFileInfo, _ := upFile.Stat()
	var fileSize int64 = upFileInfo.Size()
	fileBuffer := make([]byte, fileSize)
	upFile.Read(fileBuffer)

	response, err := s3.New(session).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(bucket),
		Key:                aws.String(key),
		ACL:                aws.String("private"),
		Body:               bytes.NewReader(fileBuffer),
		ContentLength:      aws.Int64(fileSize),
		ContentType:        aws.String(http.DetectContentType(fileBuffer)),
		ContentDisposition: aws.String("attachment"),
		//ServerSideEncryption: aws.String("AES256"),
	})

	versionID := *response.VersionId
	etag := strings.ToLower(*response.ETag)
	return versionID, etag, err
}

func S3Delete(bucket string, filepath string) (err error) {
	s := S3Connect()
	svc := s3.New(s)
	// Setup BatchDeleteIterator to iterate through a list of objects.
	iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
		Bucket: aws.String(bucket), Prefix: aws.String(filepath),
	})
	// Traverse iterator deleting each object
	if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
		exitErrorf("Unable to delete objects from bucket %q, %v", bucket, err)
	}
	fmt.Printf("Deleted object(s) from bucket: %s", bucket)
	return err
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

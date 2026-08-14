package s3storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func S3BucketVersioning(bucketname string) {
	s3_endpoint := os.Getenv("S3_CONNECT")
	s3_endpoint = strings.TrimPrefix(s3_endpoint, "http://")
	s3_acceskeyid := os.Getenv("S3_ACCESSKEYID")
	s3_secretacceskeyid := os.Getenv("S3_SECRETACCESSKEYID")
	s3_region := os.Getenv("S3_REGION")

	s3Client, err := minio.New(s3_endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3_acceskeyid, s3_secretacceskeyid, ""),
		Secure: false,
		Region: s3_region,
	})
	if err != nil {
		log.Fatalln(err)
	}
	err = s3Client.EnableVersioning(context.Background(), bucketname)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(bucketname + " Bucket Versioning Enabled")
}

func S3ObjectCount(bucketname string) (count int) {
	s3_endpoint := os.Getenv("S3_CONNECT")
	s3_endpoint = strings.TrimPrefix(s3_endpoint, "http://")
	s3_acceskeyid := os.Getenv("S3_ACCESSKEYID")
	s3_secretacceskeyid := os.Getenv("S3_SECRETACCESSKEYID")
	s3_region := os.Getenv("S3_REGION")

	s3Client, err := minio.New(s3_endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3_acceskeyid, s3_secretacceskeyid, ""),
		Secure: false,
		Region: s3_region,
	})
	if err != nil {
		log.Fatalln(err)
	}
	opts := minio.ListObjectsOptions{
		UseV1:     true,
		Recursive: true,
	}
	counter := 0
	// List all objects from a bucket-name with a matching prefix.
	for object := range s3Client.ListObjects(context.Background(), bucketname, opts) {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		counter++
		//fmt.Println(object)
	}
	return counter
}

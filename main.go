package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func handler(_ context.Context, s3Event events.S3Event) {
	record := s3Event.Records[0]
	key := record.S3.Object.Key
	sess, _ := session.NewSession(&aws.Config{Region: &record.AWSRegion})

	//Download file to a temporary folder
	downloader := s3manager.NewDownloader(sess)
	file, err := os.Create(fmt.Sprintf("/tmp/%s", key))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: &record.S3.Bucket.Name,
			Key:    &key,
		})
	if err != nil {
		panic(err)
	}
	log.Printf("Downloaded %s", file.Name())

	//transform wav file to a compress mp3 file
	outputFile := strings.Replace(file.Name(), filepath.Ext(file.Name()), ".mp3", 1)
	cmd := exec.Command("ffmpeg", "-i", file.Name(), outputFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	log.Printf("Execution output:\n%s\n", string(out))
	output, err := os.Open(outputFile)
	if err != nil {
		panic(err)
	}

	//put mp3 file in converted bucket
	destinationBucket := "mgm-mp3-files"
	_, err = s3.New(sess).PutObject(&s3.PutObjectInput{
		Bucket: &destinationBucket,
		Key:    aws.String(filepath.Base(outputFile)),
		Body:   output,
	})
	log.Printf("Copied %s to %s", outputFile, record.S3.Bucket.Name)
}

func main() {
	lambda.Start(handler)
}

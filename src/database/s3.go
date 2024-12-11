package database

import (
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// -------------- Globals --------------

var endpoint = os.Getenv("S3_API_URL")
var accessKeyID = os.Getenv("S3_ACCESS_KEY")
var secretAccessKey = os.Getenv("S3_SECRET_KEY")
var useSSL = os.Getenv("S3_USE_SSL") == "true"
var S3_FILE_URI = func() string {
	proto := "http://"
	if useSSL {
		proto = "https://"
	}
	return proto + endpoint
}()

// -------------- Functions --------------

// GetS3 Get minio S3 client
func GetS3() *minio.Client {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal("Unable to create S3 client:", err)
	}
	return minioClient
}

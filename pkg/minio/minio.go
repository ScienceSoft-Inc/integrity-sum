package minio

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	MsgFailedInitiateClient string = "failed to initiate MinIO client: %w"
	MsgFailedUpload         string = "failed to upload object: %w"
)

func init() {
	fsMinIO := pflag.NewFlagSet("minio", pflag.ExitOnError)
	fsMinIO.Bool("minio-enabled", false, "enable MinIO")
	fsMinIO.String("minio-host", "minio.svc.local:9001", "MinIO host")

	viper.BindEnv("minio-access-key", "MINIO_SERVER_USER")
	viper.BindEnv("minio-secret-key", "MINIO_SERVER_PASSWORD")

	pflag.CommandLine.AddFlagSet(fsMinIO)
	if err := viper.BindPFlags(fsMinIO); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// NewMinIOClient returns the MinIO client
func NewMinIOClient(host string, log *logrus.Logger) (*minio.Client, error) {
	accessKeyID := viper.GetString("minio-access-key")
	secretAccessKey := viper.GetString("minio-secret-key")
	useSSL := false
	// log.WithFields(logrus.Fields{
	// 	"accessKeyID":     accessKeyID,
	// 	"secretAccessKey": secretAccessKey,
	// }).Debug("MinIO credentials")

	log.Debug("initializing MinIO client")
	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf(MsgFailedInitiateClient, err)
	}
	log.Debug("MinIO client initialized")
	return client, nil
}

type Storage struct {
	client *minio.Client
	log    *logrus.Logger
}

func NewStorage(log *logrus.Logger) (*Storage, error) {
	client, err := NewMinIOClient(viper.GetString("minio-host"), log)
	if err != nil {
		return nil, err
	}
	return &Storage{
		client: client,
		log:    log,
	}, nil
}

const BucketName = "integrity"

// Save stores @data into the bucket with the given @objectName
func (s *Storage) Save(ctx context.Context, objectName string, data []byte) error {
	r := bytes.NewReader(data)
	info, err := s.client.PutObject(
		ctx,
		BucketName,
		objectName,
		r,
		r.Size(),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return fmt.Errorf(MsgFailedUpload, err)
	}
	s.log.WithFields(logrus.Fields{
		"objectName": objectName,
		"size":       info.Size,
	}).Info("uploaded successfully")
	return nil
}

/*
read
object, err := minioClient.GetObject(context.Background(), "mybucket", "myobject", minio.GetObjectOptions{})
if err != nil {
    fmt.Println(err)
    return
}
defer object.Close()
*/

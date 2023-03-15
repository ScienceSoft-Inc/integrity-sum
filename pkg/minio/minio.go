package minio

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	MsgFailedInitiateClient string = "failed to initiate MinIO client: %w"
	MsgFailedUpload         string = "failed to upload object: %w"
	MsgFailedLoad           string = "failed to load object: %w"
	MsgFailedGetInfo        string = "failed to get info for object: %w"
	MsgFailedCreateBucket   string = "failed to create bucket: %w"
)

func init() {
	fsMinIO := pflag.NewFlagSet("minio", pflag.ExitOnError)
	fsMinIO.Bool("minio-enabled", false, "enable MinIO")
	fsMinIO.String("minio-host", "minio.minio.svc.cluster.local:9000", "MinIO host")
	fsMinIO.String("minio-bucket", BucketName, "MinIO bucket name")

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
	log.WithFields(logrus.Fields{
		"accessKeyID": accessKeyID,
		// 	"secretAccessKey": secretAccessKey,
	}).Debug("MinIO credentials")

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

var (
	instance *Storage
	once     sync.Once
)

func Instance() *Storage {
	return instance
}

func NewStorage(log *logrus.Logger) (*Storage, error) {
	var err error
	once.Do(func() {
		var client *minio.Client
		client, err = NewMinIOClient(viper.GetString("minio-host"), log)
		if err != nil {
			return
		}
		instance = &Storage{
			client: client,
			log:    log,
		}
	})
	return instance, nil
}

const BucketName = "integrity"

// Save stores @data into the @bucketName with the given @objectName
func (s *Storage) Save(ctx context.Context, bucketName, objectName string, data []byte) error {
	r := bytes.NewReader(data)
	info, err := s.client.PutObject(
		ctx,
		bucketName,
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
	}).Debug("uploaded successfully")
	return nil
}

// Load loads and returns data from the @bucketName for the @objectName
func (s *Storage) Load(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	r, err := s.client.GetObject(
		ctx,
		bucketName,
		objectName,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf(MsgFailedLoad, err)
	}
	defer r.Close()

	info, err := r.Stat()
	if err != nil {
		return nil, fmt.Errorf(MsgFailedGetInfo, err)
	}
	s.log.WithFields(logrus.Fields{
		"objectName": info.Key,
		"size":       info.Size,
	}).Debug("loaded successfully")
	return ioutil.ReadAll(r)
}

// Creates a new bucket with the given @bucketName if it not exists
func (s *Storage) CreateBucketIfNotExists(ctx context.Context, bucketName string) error {
	// check the bucket is exist
	isExist, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf(MsgFailedCreateBucket, err)
	}
	if isExist {
		return nil
	}

	// create new one
	err = s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf(MsgFailedCreateBucket, err)
	}
	s.log.WithFields(logrus.Fields{
		"bucketName": bucketName,
	}).Debug("created successfully")

	return nil
}

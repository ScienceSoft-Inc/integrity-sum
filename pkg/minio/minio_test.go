package minio

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	cleanup := setup()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func generatePassword(length int) string {
	const passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)
	rand.Seed(time.Now().UnixNano())
	for i := range password {
		password[i] = passwordCharset[rand.Intn(len(passwordCharset))]
	}
	return string(password)
}

func setup() func() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		logrus.Fatalf("could not construct pool: %s", err)
	}
	// uses pool to try to connect to the Docker
	err = pool.Client.Ping()
	if err != nil {
		logrus.Fatalf("could not connect to Docker: %s", err)
	}

	// test credentials
	testUser := generatePassword(6)
	testPassword := generatePassword(12)

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("bitnami/minio", "latest", []string{
		"MINIO_ROOT_USER=" + testUser,
		"MINIO_ROOT_PASSWORD=" + testPassword,
		"MINIO_DEFAULT_BUCKETS=bucket",
		"MINIO_SERVER_HOST=minio",
	})
	if err != nil {
		logrus.Fatalf("could not start resource: %s", err)
	}

	// connection to the MinIO server
	host := "127.0.0.1"
	viper.Set("minio-host", host+":"+resource.GetPort("9000/tcp"))
	viper.Set("minio-access-key", testUser)
	viper.Set("minio-secret-key", testPassword)

	const dockerTimeOutSeconds = 10
	resource.Expire(uint(dockerTimeOutSeconds))
	pool.MaxWait = dockerTimeOutSeconds * time.Second
	waitMinioServiceStarted(pool, "http://"+host+":"+resource.GetPort("9001/tcp")+"/")

	return func() {
		if err := pool.Purge(resource); err != nil {
			logrus.Errorf("could not purge docker pool resource: %s", err)
		}
	}
}

func waitMinioServiceStarted(pool *dockertest.Pool, addr string) {
	if err := pool.Retry(func() error {
		logrus.Infof("waiting for MinIO container, connecting %s ...", addr)
		resp, err := http.Get(addr)
		if err != nil {
			logrus.Info("container not ready, waiting...")
			return err
		}
		defer resp.Body.Close()
		return nil
	}); err != nil {
		logrus.Fatalf("could not connect to MinIO server: %s", err)
	}
	logrus.Info("the MinIO container ready")
}

func TestCreateBucket(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	log.Debugf("MinIO host: %v", viper.GetString("minio-host"))
	m, err := NewStorage(log)
	assert.NoError(t, err, "cannot create MinIO client: %v", err)
	assert.NotNil(t, m, "cannot create MinIO client")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	log.Debug("test MinIO storage: create bucket")
	err = Instance().CreateBucketIfNotExists(ctx, "test-bucket")
	assert.NoError(t, err, "cannot create bucket: %v", err)
}

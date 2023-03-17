package minio

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var log = logrus.New()

func TestMain(m *testing.M) {
	log.SetLevel(logrus.DebugLevel)
	cleanup, err := setup()
	if err != nil {
		os.Exit(1)
	}

	var code = 1
	defer func(intP *int) {
		cleanup()
		log.Debugf("exit code: %d", *intP)
		os.Exit(*intP)
	}(&code)

	_, err = NewStorage(log)
	if err != nil {
		log.Errorf("could not create storage: %s", err)
		return
	}

	code = m.Run()
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

var setupBuckets = []string{
	"my-first-bucket",
	"my-second-bucket",
}

func setup() (func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Errorf("could not construct pool: %s", err)
		return nil, err
	}
	// uses pool to try to connect to the Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Errorf("could not connect to Docker: %s", err)
		return nil, err
	}

	// test credentials
	testUser := generatePassword(6)
	testPassword := generatePassword(12)

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("bitnami/minio", "latest", []string{
		"MINIO_DEFAULT_BUCKETS=" + strings.Join(setupBuckets, ","),
		"MINIO_ROOT_USER=" + testUser,
		"MINIO_ROOT_PASSWORD=" + testPassword,
		"MINIO_SERVER_HOST=minio",
	})
	if err != nil {
		log.Errorf("could not start container: %s", err)
		return nil, err
	}

	cleanup := func() {
		if err := pool.Purge(resource); err != nil {
			log.Errorf("could not purge docker pool resource: %s", err)
		}
	}

	// connection to the MinIO server
	host := "127.0.0.1"
	viper.Set("minio-host", host+":"+resource.GetPort("9000/tcp"))
	viper.Set("minio-access-key", testUser)
	viper.Set("minio-secret-key", testPassword)

	const dockerTimeOutSeconds = 20
	resource.Expire(uint(dockerTimeOutSeconds))
	pool.MaxWait = dockerTimeOutSeconds * time.Second

	if err = waitMinioServiceStart(pool, "http://"+host+":"+resource.GetPort("9001/tcp")); err != nil {
		cleanup()
		return nil, err
	}

	printLog(pool.Client, resource.Container.ID)
	return cleanup, nil
}

func printLog(client *docker.Client, containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	buf := bytes.NewBuffer(nil)

	tNow := time.Now()
	logOptions := docker.LogsOptions{
		Context:           ctx,
		Container:         containerID,
		OutputStream:      buf,
		ErrorStream:       buf,
		InactivityTimeout: 0,
		Tail:              "100",
		Since:             0,
		Follow:            false,
		Stdout:            true,
		Stderr:            true,
		Timestamps:        false,
		RawTerminal:       false,
	}

	err := client.Logs(logOptions)
	if err != nil {
		log.Errorf("could not get logs from container (%v): %s", time.Since(tNow), err)
		return
	}

	log.Debugf("container logs (%v):", time.Since(tNow))
	printOutput(buf.String())
}

func printOutput(str string) {
	formatter := log.Formatter
	defer func() { log.Formatter = formatter }()

	log.Formatter = &logrus.TextFormatter{DisableQuote: true}
	log.Debug(str)
}

func waitMinioServiceStart(pool *dockertest.Pool, addr string) error {
	if err := pool.Retry(func() error {
		log.Infof("waiting for MinIO container, connecting %s ...", addr)
		resp, err := http.Get(addr)
		if err != nil {
			log.Debugf("container not ready: %v", err)
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK")
		}
		defer resp.Body.Close()
		return nil
	}); err != nil {
		log.Errorf("could not connect to MinIO server: %s", err)
		return err
	}

	// important: waiting for MinIO setup to be done
	time.Sleep(7 * time.Second) // TODO: check the log for "setup completed"

	log.Debugf("the MinIO container is ready now")
	return nil
}

func TestSetupBuckets(t *testing.T) {
	ctxLog, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	buckets, err := Instance().ListBuckets(ctxLog)
	assert.NoError(t, err, "cannot list buckets: %v", err)
	for i, v := range buckets {
		assert.Equal(t, setupBuckets[i], v.Name, "buckets not equal: %v-%v", setupBuckets[i], v.Name)
	}
}

func TestCreateBucket(t *testing.T) {
	ctxLog, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	log.Debug("test MinIO storage: create bucket")
	err := Instance().CreateBucketIfNotExists(ctxLog, "test-bucket")
	assert.NoError(t, err, "cannot create bucket: %v", err)
}

func TestSaveLoadObject(t *testing.T) {
	const (
		testMsg = "test-data"
		testObj = "test-object"
	)
	ctxLog, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := Instance().CreateBucketIfNotExists(ctxLog, defaultBucketName)
	assert.NoError(t, err, "cannot create bucket: %v", err)

	err = Instance().Save(ctxLog, defaultBucketName, testObj, []byte(testMsg))
	assert.NoError(t, err, "cannot save data: %v", err)

	data, err := Instance().Load(ctxLog, defaultBucketName, testObj)
	assert.NoError(t, err, "cannot load data: %v", err)
	assert.Equal(t, testMsg, string(data))
}

package initialize

import (
	"context"
	"fmt"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/connection"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/repositories"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/services"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts"
	splunkclient "github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts/splunk"
)

func Initialize(ctx context.Context, logger *logrus.Logger) {
	// Initialize database
	db, err := connection.ConnectionToDB(logger)
	if err != nil {
		logger.Fatalf("can't connect to database: %s", err)
	}

	// Initialize repository
	repository := repositories.NewAppRepository(logger, db)
	repositoryHashStorage := repositories.NewHashStorageRepository(logger, db)
	repositoryReleaseStorage := repositories.NewReleaseStorageRepository(logger, db)

	// Create alert sender
	splunkUrl := viper.GetString("splunk-url")
	splunkToken := viper.GetString("splunk-token")
	splunkInsecureSkipVerify := viper.GetBool("splunk-insecure-skip-verify")
	var alertsSender alerts.Sender
	if len(splunkUrl) > 0 && len(splunkToken) > 0 {
		alertsSender = splunkclient.New(logger, splunkUrl, splunkToken, splunkInsecureSkipVerify)
	}

	// Initialize service
	algorithm := viper.GetString("algorithm")

	serviceReleaseStorage := services.NewReleaseStorageService(repositoryReleaseStorage, algorithm, logger)
	serviceHashStorage := services.NewHashStorageService(repositoryHashStorage, algorithm, logger)
	service := services.NewAppService(repository, alertsSender, serviceReleaseStorage, serviceHashStorage, algorithm, logger)

	// Initialize kubernetesAPI
	dataFromK8sAPI, err := service.GetDataFromK8sAPI()
	if err != nil {
		logger.Fatalf("can't get data from K8sAPI: %s", err)
	}

	//Getting pid
	procName := viper.GetString("process")
	pid, err := service.GetPID(procName)
	if err != nil {
		logger.Fatalf("err while getting pid %s", err)
	}
	if pid == 0 {
		logger.Fatalf("proc with name %s not exist", procName)
	}

	//Getting the path to the monitoring directory
	dirPath := fmt.Sprintf("/proc/%d/root/%s", pid, viper.GetString("monitoring-path"))

	ticker := time.NewTicker(viper.GetDuration("duration-time"))

	// Initialize сlearing
	logger.Info("Сlearing the database of old data")
	err = repositoryReleaseStorage.DeleteOldData()

	var wg sync.WaitGroup
	wg.Add(1)
	go func(ctx context.Context, ticker *time.Ticker) {
		defer wg.Done()
		for {
			if !service.IsExistDeploymentNameInDB(dataFromK8sAPI.KuberData.TargetName) {
				logger.Info("Deployment name does not exist in database, save data")
				err = service.Start(ctx, dirPath, dataFromK8sAPI.DeploymentData)
				if err != nil {
					logger.Fatalf("Error when starting to get and save hash data %s", err)
				}

				if err != nil {
					logger.Errorf("Error when clearing the database of old data %s", err)
				}
			} else {
				logger.Info("Deployment name exists in database, checking data")
				for range ticker.C {
					err = service.Check(ctx, dirPath, dataFromK8sAPI.DeploymentData, dataFromK8sAPI.KuberData)
					if err != nil {
						logger.Fatalf("Error when starting to check hash data %s", err)
					}
					logger.Info("Check completed")
				}
			}
		}
	}(ctx, ticker)
	wg.Wait()
	ticker.Stop()
}

package initialize

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/services"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/repositories"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts"
	splunkclient "github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts/splunk"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts/syslog"
)

func Initialize(ctx context.Context, logger *logrus.Logger) {
	// Initialize database
	db, err := repositories.ConnectionToDB(logger)
	if err != nil {
		logger.Fatalf("can't connect to database: %s", err)
	}

	// Initialize repository
	repository := repositories.NewAppRepository(logger, db)

	// Create alert sender
	splunkUrl := viper.GetString("splunk-url")
	splunkToken := viper.GetString("splunk-token")
	splunkInsecureSkipVerify := viper.GetBool("splunk-insecure-skip-verify")

	alertsRegistry := alerts.Registry{}
	if len(splunkUrl) > 0 && len(splunkToken) > 0 {
		alertsRegistry.Add(alerts.TypeSplunk, splunkclient.New(logger, splunkUrl, splunkToken, splunkInsecureSkipVerify))
	}

	if viper.GetBool("syslog-enabled") {
		addr := fmt.Sprintf("%s:%d", viper.GetString("syslog-host"), viper.GetInt("syslog-port"))
		syslogSender, err := syslog.New(logger, viper.GetString("syslog-proto"), addr, syslog.DefaultPriority)
		if err != nil {
			logger.Fatalf("cannnot initialize sysylog client %v", err)
		}
		alertsRegistry.Add(alerts.TypeSyslog, syslogSender)
		logger.Info("notification to syslog enabled")

		defer syslogSender.Close()
	}

	// Initialize service
	algorithm := viper.GetString("algorithm")

	service := services.NewAppService(repository, alertsRegistry, algorithm, logger)

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

	var wg sync.WaitGroup
	wg.Add(1)
	go func(ctx context.Context, ticker *time.Ticker) {
		defer wg.Done()
		for {
			if !service.IsExistDeploymentNameInDB(dataFromK8sAPI.KubeData.TargetName) {
				logger.Info("Deployment name does not exist in database, save data")
				err = service.Start(ctx, dirPath, dataFromK8sAPI.DeploymentData)
				if err != nil {
					logger.Fatalf("Error when starting to get and save hash data %s", err)
				}
			} else {
				logger.Info("Deployment name exists in database, checking data")
				for range ticker.C {
					err = service.Check(ctx, dirPath, dataFromK8sAPI.DeploymentData, dataFromK8sAPI.KubeData)
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

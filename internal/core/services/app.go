package services

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/models"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/ports"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/api"
)

type AppService struct {
	ports.IHashService
	ports.IAppRepository
	ports.IKuberService
	alertSender alerts.Sender
	logger      *logrus.Logger
}

// NewAppService creates a new struct AppService
func NewAppService(r ports.IAppRepository, alertSender alerts.Sender, algorithm string, logger *logrus.Logger) *AppService {
	return &AppService{
		IHashService:   NewHashService(r, strings.ToUpper(algorithm), logger),
		IAppRepository: r,
		IKuberService:  NewKubeService(logger),
		alertSender:    alertSender,
		logger:         logger,
	}
}

// GetPID returns process PID by name
func (as *AppService) GetPID(procName string) (pid int, err error) {
	cmdOut, err := exec.Command("pidof", procName).Output()
	if err != nil {
		as.logger.WithField("procName", procName).WithError(err).Error("GetPID(): proc name not found")
		return
	}
	// if found, we always have a list of PIDs
	ss := strings.Split(string(cmdOut), " ")
	pid, err = strconv.Atoi(strings.TrimSpace(ss[0]))
	return
}

// LaunchHasher takes a path to a directory and returns HashData
func (as *AppService) LaunchHasher(ctx context.Context, dirPath string) []*api.HashData {
	jobs := make(chan string)
	results := make(chan *api.HashData)
	go as.IHashService.WorkerPool(jobs, results)
	go api.SearchFilePath(ctx, dirPath, jobs, as.logger)
	allHashData := api.Result(ctx, results)

	return allHashData
}

// IsExistDeploymentNameInDB checks if the database is empty
func (as *AppService) IsExistDeploymentNameInDB(deploymentName string) bool {
	isEmptyDB, err := as.IAppRepository.IsExistDeploymentNameInDB(deploymentName)
	if err != nil && err != sql.ErrNoRows {
		as.logger.Fatalf("database check error %s", err)
	}
	return isEmptyDB
}

// Start getting the hash sum of all files, outputs to os.Stdout and saves to the database
func (as *AppService) Start(ctx context.Context, dirPath string, deploymentData *models.DeploymentData) error {
	allHashData := as.LaunchHasher(ctx, dirPath)
	err := as.IHashService.SaveHashData(allHashData, deploymentData)
	if err != nil {
		as.logger.Error("Error save hash data to database ", err)
		return err
	}

	return nil
}

// Check getting the hash sum of all files, matches them and outputs to os.Stdout changes
func (as *AppService) Check(ctx context.Context, dirPath string, deploymentData *models.DeploymentData, kuberData *models.KubeData) error {
	hashDataCurrentByDirPath := as.LaunchHasher(ctx, dirPath)

	dataFromDBbyPodName, err := as.IHashService.GetHashData(dirPath, deploymentData)
	if err != nil {
		as.logger.Error("Error getting hash data from database ", err)
		return err
	}

	isDataChanged := as.IHashService.IsDataChanged(hashDataCurrentByDirPath, dataFromDBbyPodName, deploymentData)
	if isDataChanged {

		// alert sender is optional
		if as.alertSender != nil {
			err := as.alertSender.Send(alerts.Alert{
				Time:    time.Now(),
				Message: fmt.Sprintf("Restart deployment %v", deploymentData.NameDeployment),
				Reason:  "mismatch file content",
				Path:    dirPath,
			})
			if err != nil {
				as.logger.WithError(err).Error("Failed send alert")
			}
		}

		err = as.IHashService.DeleteFromTable(deploymentData.NameDeployment)
		if err != nil {
			as.logger.Error("Error while deleting rows in database", err)
			return err
		}

		err = as.IKuberService.RolloutDeployment(kuberData)
		if err != nil {
			as.logger.Error("Error while rolling out deployment in k8s", err)
			return err
		}
	}
	return nil
}

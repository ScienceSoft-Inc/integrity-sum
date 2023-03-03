package services

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/models"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/ports"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/api"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/hasher"
)

// TODO: the purpose of this service is...?
type HashService struct {
	Repository ports.IAppRepository // TODO: why do we need it here?
	alg        string
	logger     *logrus.Logger
}

// NewHashService creates a new struct HashService
func NewHashService(Repository ports.IAppRepository, alg string, logger *logrus.Logger) *HashService {
	return &HashService{
		Repository: Repository,
		alg:        alg,
		logger:     logger,
	}
}

// WorkerPool launches a certain number of workers for concurrent processing
func (hs HashService) WorkerPool(jobs chan string, results chan *api.HashData) {
	countWorkers := viper.GetInt("count-workers")

	var wg sync.WaitGroup
	for w := 1; w <= countWorkers; w++ {
		wg.Add(1) // TODO: countWorkers
		go hs.Worker(&wg, jobs, results)
	}
	defer close(results)
	wg.Wait()
}

// Worker gets jobs from a pipe and writes the result to stdout and database
func (hs HashService) Worker(wg *sync.WaitGroup, jobs <-chan string, results chan<- *api.HashData) {
	defer wg.Done() // TODO: not in place
	for j := range jobs {
		data, err := hs.CreateHash(j)
		if err != nil {
			hs.logger.Errorf("error creating file hash - %s, %s", j, err)
			continue
		}
		results <- data
	}
}

// TODO: is it need now?
// CreateHash creates a new object with a hash sum
func (hs HashService) CreateHash(fileName string) (*api.HashData, error) {
	hash, err := hasher.NewFileHasher(hs.alg, hs.logger).HashFile(fileName)
	if err != nil {
		hs.logger.WithError(err).WithField("fileName", fileName).Error("hashing file")
		return nil, err
	}
	return &api.HashData{
		Hash:         hash,
		FileName:     filepath.Base(fileName),
		FullFilePath: fileName,
		Algorithm:    hs.alg,
	}, nil
}

// SaveHashData accesses the repository to save data to the database
func (hs HashService) SaveHashData(allHashData []*api.HashData, deploymentData *models.DeploymentData) error {
	err := hs.Repository.SaveHashData(allHashData, deploymentData)
	if err != nil {
		hs.logger.Error("error while saving data to database", err)
		return err
	}
	return nil
}

// GetHashData accesses the repository to get data from the database
func (hs HashService) GetHashData(dirFiles string, deploymentData *models.DeploymentData) ([]*models.HashDataFromDB, error) {
	hashData, err := hs.Repository.GetHashData(dirFiles, hs.alg, deploymentData)
	if err != nil {
		hs.logger.Error("hashData service didn't get hashData sum", err)
		return nil, err
	}
	return hashData, nil
}

func (hs HashService) DeleteFromTable(nameDeployment string) error {
	err := hs.Repository.DeleteFromTable(nameDeployment)
	if err != nil {
		hs.logger.Error("err while deleting rows in database", err)
		return err
	}
	return nil
}

// IsDataChanged checks if the current data has changed with the data stored in the database
func (hs HashService) IsDataChanged(currentHashData []*api.HashData, hashDataFromDB []*models.HashDataFromDB, deploymentData *models.DeploymentData) bool {
	isDataChanged := wasDataChanged(hashDataFromDB, currentHashData, deploymentData)
	isAddedFiles := wasAddedFiles(currentHashData, hashDataFromDB)

	if isDataChanged || isAddedFiles {
		return true
	}
	return false
}

func wasDataChanged(hashSumFromDB []*models.HashDataFromDB, currentHashData []*api.HashData, deploymentData *models.DeploymentData) bool {
	for _, dataFromDB := range hashSumFromDB {
		trigger := false
		for _, dataCurrent := range currentHashData {
			if dataFromDB.FullFilePath == dataCurrent.FullFilePath && dataFromDB.Algorithm == dataCurrent.Algorithm {
				if dataFromDB.Hash != dataCurrent.Hash {
					fmt.Printf("Changed: file - %s the path %s, old hash sum %s, new hash sum %s\n",
						dataFromDB.FileName, dataFromDB.FullFilePath, dataFromDB.Hash, dataCurrent.Hash)
					return true
				}
				if dataFromDB.ImageContainer != deploymentData.Image && dataFromDB.NameDeployment == deploymentData.NameDeployment {
					fmt.Printf("Changed image container: file - %s the path %s, old image %s, new image %s\n",
						dataFromDB.FileName, dataFromDB.FullFilePath, dataFromDB.ImageContainer, deploymentData.Image)
					return true
				}
				trigger = true
				break
			}
		}

		if !trigger {
			fmt.Printf("Deleted: file - %s the path %s hash sum %s\n", dataFromDB.FileName, dataFromDB.FullFilePath, dataFromDB.Hash)
			return true
		}
	}
	return false
}

func wasAddedFiles(currentHashData []*api.HashData, hashDataFromDB []*models.HashDataFromDB) bool {
	dataFromDB := make(map[string]struct{}, len(hashDataFromDB))
	for _, value := range hashDataFromDB {
		dataFromDB[value.FullFilePath] = struct{}{}
	}

	for _, dataCurrent := range currentHashData {
		if _, ok := dataFromDB[dataCurrent.FullFilePath]; !ok {
			fmt.Printf("Changed: the current data is different from the data in the database, current file - %s the path %s hash sum %s\n",
				dataCurrent.FileName, dataCurrent.FullFilePath, dataCurrent.Hash)
			return true
		}
	}
	return false
}

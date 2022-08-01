package services

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/integrity-sum/internal/core/models"
	"github.com/integrity-sum/internal/core/ports"
	"github.com/integrity-sum/pkg/api"
	"github.com/integrity-sum/pkg/hasher"

	"github.com/sirupsen/logrus"
)

type HashService struct {
	hashRepository ports.IHashRepository
	hasher         hash.Hash
	alg            string
	logger         *logrus.Logger
}

// NewHashService creates a new struct HashService
func NewHashService(hashRepository ports.IHashRepository, alg string, logger *logrus.Logger) (*HashService, error) {
	h := hasher.NewHashSummator(alg)

	return &HashService{
		hashRepository: hashRepository,
		hasher:         h,
		alg:            alg,
		logger:         logger,
	}, nil
}

// WorkerPool launches a certain number of workers for concurrent processing
func (hs HashService) WorkerPool(jobs chan string, results chan *api.HashData) {
	countWorkers, err := strconv.Atoi(os.Getenv("COUNT_WORKERS"))
	if err != nil {
		countWorkers = runtime.NumCPU()
	}

	var wg sync.WaitGroup
	for w := 1; w <= countWorkers; w++ {
		wg.Add(1)
		go hs.Worker(&wg, jobs, results)
	}
	defer close(results)
	wg.Wait()
}

// Worker gets jobs from a pipe and writes the result to stdout and database
func (hs HashService) Worker(wg *sync.WaitGroup, jobs <-chan string, results chan<- *api.HashData) {
	defer wg.Done()
	for j := range jobs {
		data, err := hs.CreateHash(j)
		if err != nil {
			hs.logger.Errorf("error creating file hash - %s, %s", j, err)
			continue
		}

		results <- data
	}
}

// CreateHash creates a new object with a hash sum
func (hs HashService) CreateHash(path string) (*api.HashData, error) {
	file, err := os.Open(path)
	if err != nil {
		hs.logger.Error("not exist file path ", err)
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			hs.logger.Error(fmt.Sprintf("[HashService]Error closing file: %s", err))
		}
	}(file)

	_, err = io.Copy(hs.hasher, file)
	if err != nil {
		return nil, err
	}
	res := hex.EncodeToString(hs.hasher.Sum(nil))
	defer hs.hasher.Reset()

	if err != nil {
		hs.logger.Error("not got hash sum ", err)
		return nil, err
	}

	outputHashSum := api.HashData{
		Hash:         res,
		FileName:     filepath.Base(path),
		FullFilePath: path,
		Algorithm:    hs.alg,
	}

	return &outputHashSum, nil
}

// SaveHashData accesses the repository to save data to the database
func (hs HashService) SaveHashData(allHashData []*api.HashData, deploymentData *models.DeploymentData) error {
	err := hs.hashRepository.SaveHashData(allHashData, deploymentData)
	if err != nil {
		hs.logger.Error("error while saving data to database", err)
		return err
	}
	return nil
}

// GetHashData accesses the repository to get data from the database
func (hs HashService) GetHashData(dirFiles string, deploymentData *models.DeploymentData) ([]*models.HashDataFromDB, error) {
	h, err := hs.hashRepository.GetHashData(dirFiles, hs.alg, deploymentData)
	if err != nil {
		hs.logger.Error("hash service didn't get hash sum", err)
		return nil, err
	}

	return h, nil
}

func (hs HashService) DeleteFromTable(nameDeployment string) error {
	err := hs.hashRepository.DeleteFromTable(nameDeployment)
	if err != nil {
		hs.logger.Error("err while deleting rows in database", err)
		return err
	}
	return nil
}

// IsDataChanged checks if the current data has changed with the data stored in the database
func (hs HashService) IsDataChanged(currentHashData []*api.HashData, hashDataFromDB []*models.HashDataFromDB, deploymentData *models.DeploymentData) (bool, error) {
	isDataChanged := wasDataChanged(hashDataFromDB, currentHashData, deploymentData)
	isAddedFiles := wasAddedFiles(currentHashData, hashDataFromDB)

	if isDataChanged || isAddedFiles {
		return true, nil
	}
	return false, nil
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

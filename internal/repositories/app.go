package repositories

import (
	"database/sql"

	"github.com/sirupsen/logrus"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/models"
)

type AppRepository struct {
	logger *logrus.Logger
	db     *sql.DB
}

func NewAppRepository(logger *logrus.Logger, db *sql.DB) *AppRepository {
	return &AppRepository{
		logger: logger,
		db:     db,
	}
}

// IsExistDeploymentNameInDB checks if the base is empty
func (ar AppRepository) IsExistDeploymentNameInDB(deploymentName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM hashfiles WHERE name_deployment=$1;`
	err := ar.db.QueryRow(query, deploymentName).Scan(&count)
	if err != nil {
		ar.logger.Error("err while scan row in database ", err)
		return false, err
	}
	if count == 0 {
		ar.logger.Info("no rows in database")
		return false, nil
	}
	return true, nil
}

// GetHashData retrieves data from the database using the path and algorithm
func (ar AppRepository) GetHashData(dirFiles, algorithm string, deploymentData *models.DeploymentData) ([]*models.HashDataFromDB, error) {
	var allHashDataFromDB []*models.HashDataFromDB

	query := `SELECT id,file_name,full_file_path,hash_sum,algorithm,image_tag,name_pod,name_deployment
		FROM hashfiles WHERE full_file_path LIKE $1 and algorithm=$2 and name_pod=$3`

	rows, err := ar.db.Query(query, "%"+dirFiles+"%", algorithm, deploymentData.NamePod)
	if err != nil {
		ar.logger.Error("err while getting data from database ", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var hashDataFromDB models.HashDataFromDB
		err = rows.Scan(&hashDataFromDB.ID, &hashDataFromDB.FileName, &hashDataFromDB.FullFilePath,
			&hashDataFromDB.Hash, &hashDataFromDB.Algorithm, &hashDataFromDB.ImageContainer,
			&hashDataFromDB.NamePod, &hashDataFromDB.NameDeployment)
		if err != nil {
			ar.logger.Error("err while scan data from database ", err)
			return nil, err
		}
		allHashDataFromDB = append(allHashDataFromDB, &hashDataFromDB)
	}

	return allHashDataFromDB, nil
}

// DeleteFromTable removes data from the table that matches the name of the deployment
func (ar AppRepository) DeleteFromTable(nameDeployment string) error {
	query := `DELETE FROM hashfiles WHERE name_deployment=$1;`
	_, err := ar.db.Exec(query, nameDeployment)
	if err != nil {
		ar.logger.Error("err while deleting rows in database", err)
		return err
	}
	return nil
}

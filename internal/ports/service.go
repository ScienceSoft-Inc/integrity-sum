package ports

import (
	"github.com/ScienceSoft-Inc/integrity-sum/internal/models"
	"k8s.io/client-go/kubernetes"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_service.go

type IKuberService interface {
	Connect() (*kubernetes.Clientset, error)
	GetDataFromK8sAPI() (*models.DataFromK8sAPI, error)
	GetKubeData() (*models.KubeData, error)
	GetDataFromDeployment(kuberData *models.KubeData) (*models.DeploymentData, error)
	RolloutDeployment(kuberData *models.KubeData) error
}

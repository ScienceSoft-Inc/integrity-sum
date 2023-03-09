package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/models"
)

type KubeClient struct {
	logger    *logrus.Logger
	clientset *kubernetes.Clientset
}

// NewKubeService creates a new service for working with the Kubernetes API
func NewKubeService(logger *logrus.Logger) *KubeClient {
	return &KubeClient{
		logger: logger,
	}
}

// Connect to Kubernetes API
func (ks *KubeClient) Connect() (*kubernetes.Clientset, error) {
	if ks.clientset != nil {
		return ks.clientset, nil
	}

	ks.logger.Info("### 🌀 Attempting to use in cluster config")
	config, err := rest.InClusterConfig()
	if err != nil {
		ks.logger.Error(err)
		return nil, err
	}

	ks.logger.Info("### 💻 Connecting to Kubernetes API, using host: ", config.Host)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		ks.logger.Error(err)
		return nil, err
	}

	ks.clientset = clientset

	return clientset, nil
}

// GetDataFromK8sAPI returns data from deployment
func (ks *KubeClient) GetDataFromK8sAPI() (*models.DataFromK8sAPI, error) {
	kubeData, err := ks.GetKubeData()
	if err != nil {
		ks.logger.Errorf("can't connect to K8sAPI: %s", err)
		return nil, err
	}
	deploymentData, err := ks.GetDataFromDeployment(kubeData)
	if err != nil {
		ks.logger.Errorf("error while getting data from kuberAPI %s", err)
		return nil, err
	}

	if err != nil {
		ks.logger.Errorf("err while getting data from configMap K8sAPI %s", err)
		return &models.DataFromK8sAPI{}, err
	}

	dataFromK8sAPI := &models.DataFromK8sAPI{
		KubeData:       kubeData,
		DeploymentData: deploymentData,
	}

	return dataFromK8sAPI, nil
}

// GetKubeData returns kubeData
func (ks *KubeClient) GetKubeData() (*models.KubeData, error) {
	clientset, err := ks.Connect()
	if err != nil {
		return nil, err
	}

	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		ks.logger.Error(err)
		return nil, err
	}
	namespace := string(namespaceBytes)

	podName := os.Getenv("POD_NAME")

	targetName := func(podName string) string {
		elements := strings.Split(podName, "-")
		newElements := elements[:len(elements)-2]
		return strings.Join(newElements, "-")
	}(podName)
	if targetName == "" {
		ks.logger.Fatalln("### 💥 Env var DEPLOYMENT_NAME was not set")
	}
	targetType := os.Getenv("DEPLOYMENT_TYPE")
	kubeData := &models.KubeData{
		Clientset:  clientset,
		Namespace:  namespace,
		TargetName: targetName,
		TargetType: targetType,
	}
	return kubeData, nil
}

// GetDataFromDeployment returns data from deployment
func (ks *KubeClient) GetDataFromDeployment(kubeData *models.KubeData) (*models.DeploymentData, error) {
	allDeploymentData, err := kubeData.Clientset.AppsV1().Deployments(kubeData.Namespace).Get(context.Background(), kubeData.TargetName, metav1.GetOptions{})
	if err != nil {
		ks.logger.Error("err while getting data from kuberAPI ", err)
		return nil, err
	}

	deploymentData := &models.DeploymentData{
		NamePod:        os.Getenv("POD_NAME"),
		Timestamp:      fmt.Sprintf("%v", allDeploymentData.CreationTimestamp),
		NameDeployment: kubeData.TargetName,
	}

	for _, v := range allDeploymentData.Spec.Template.Spec.Containers {
		deploymentData.Image = v.Image
	}

	if value, ok := allDeploymentData.Annotations["meta.helm.sh/release-name"]; ok {
		deploymentData.ReleaseName = value
	}

	return deploymentData, nil
}

// RolloutDeployment rolls out deployment
func (ks *KubeClient) RolloutDeployment(kubeData *models.KubeData) error {
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := kubeData.Clientset.AppsV1().Deployments(kubeData.Namespace).Patch(context.Background(), kubeData.TargetName, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		ks.logger.Printf("### 👎 Warning: Failed to patch %v, restart failed: %v", kubeData.TargetType, err)
		return err
	} else {
		ks.logger.Printf("### ✅ Target %v, named %v was restarted!", kubeData.TargetType, kubeData.TargetName)
	}
	return nil
}

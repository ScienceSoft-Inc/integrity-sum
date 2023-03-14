// Code generated by MockGen. DO NOT EDIT.
// Source: repository.go

// Package mock_ports is a generated GoMock package.
package mock_ports

import (
	reflect "reflect"

	models "github.com/ScienceSoft-Inc/integrity-sum/internal/core/models"
	gomock "github.com/golang/mock/gomock"
)

// MockIAppRepository is a mock of IAppRepository interface.
type MockIAppRepository struct {
	ctrl     *gomock.Controller
	recorder *MockIAppRepositoryMockRecorder
}

// MockIAppRepositoryMockRecorder is the mock recorder for MockIAppRepository.
type MockIAppRepositoryMockRecorder struct {
	mock *MockIAppRepository
}

// NewMockIAppRepository creates a new mock instance.
func NewMockIAppRepository(ctrl *gomock.Controller) *MockIAppRepository {
	mock := &MockIAppRepository{ctrl: ctrl}
	mock.recorder = &MockIAppRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIAppRepository) EXPECT() *MockIAppRepositoryMockRecorder {
	return m.recorder
}

// DeleteFromTable mocks base method.
func (m *MockIAppRepository) DeleteFromTable(nameDeployment string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFromTable", nameDeployment)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFromTable indicates an expected call of DeleteFromTable.
func (mr *MockIAppRepositoryMockRecorder) DeleteFromTable(nameDeployment interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFromTable", reflect.TypeOf((*MockIAppRepository)(nil).DeleteFromTable), nameDeployment)
}

// GetHashData mocks base method.
func (m *MockIAppRepository) GetHashData(dirFiles, algorithm string, deploymentData *models.DeploymentData) ([]*models.HashDataFromDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHashData", dirFiles, algorithm, deploymentData)
	ret0, _ := ret[0].([]*models.HashDataFromDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHashData indicates an expected call of GetHashData.
func (mr *MockIAppRepositoryMockRecorder) GetHashData(dirFiles, algorithm, deploymentData interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHashData", reflect.TypeOf((*MockIAppRepository)(nil).GetHashData), dirFiles, algorithm, deploymentData)
}

// IsExistDeploymentNameInDB mocks base method.
func (m *MockIAppRepository) IsExistDeploymentNameInDB(deploymentName string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsExistDeploymentNameInDB", deploymentName)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsExistDeploymentNameInDB indicates an expected call of IsExistDeploymentNameInDB.
func (mr *MockIAppRepositoryMockRecorder) IsExistDeploymentNameInDB(deploymentName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsExistDeploymentNameInDB", reflect.TypeOf((*MockIAppRepository)(nil).IsExistDeploymentNameInDB), deploymentName)
}
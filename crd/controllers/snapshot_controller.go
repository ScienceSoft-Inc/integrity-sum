/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	integrityv1 "integrity/snapshot/api/v1"

	// mstorage "file:///home/sshliayonkin@scnsoft.com/go/src/github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
	mstorage "github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
)

// SnapshotReconciler reconciles a Snapshot object
type SnapshotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Minio  *minio.Client
}

//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Snapshot object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SnapshotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	l := log.FromContext(ctx)
	var snapshot integrityv1.Snapshot
	if err := r.Get(ctx, req.NamespacedName, &snapshot); err != nil {
		l.Error(err, "unable to fetch snapshot")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	l.Info("snapshot found", "snapshot.Spec", snapshot.Spec)

	// TODO: if status - not uploaded

	ms, err := r.minIOStorage(ctx, l)
	if err != nil {
		l.Error(err, "unable to get minio client")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// testing minIO storage:
	// buckets, err := ms.ListBuckets(ctx)
	// if err != nil {
	// 	l.Error(err, "unable to list buckets")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }
	// l.Info("buckets found", "buckets", buckets)

	// imageInfo := strings.Split(snapshot.Spec.Image, ":")
	// objName := req.NamespacedName.Namespace + "/" + imageInfo[0] + "/" + imageInfo[1]

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = ms.Save(ctx, "integrity", // TODO: mstorage.DefaultBucketName
		buildObjectName(req.NamespacedName.Namespace, snapshot.Spec.Image), // TODO: mstorage.BuildObjectName(req.NamespacedName.Namespace, snapshot.Spec.Image),
		[]byte(snapshot.Spec.Base64Hashes),
	); err != nil {
		return ctrl.Result{}, err
	}
	l.Info("snapshot saved", "snapshot.Spec", snapshot.Spec)

	// buckets, err := ms.ListBuckets(ctx)
	// if err != nil {
	// 	l.Error(err, "unable to list buckets")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }
	// l.Info("buckets found", "buckets", buckets)

	return ctrl.Result{}, nil
}

var (
	minioOnce        sync.Once
	minioInitialized bool
)

func (r *SnapshotReconciler) minIOStorage(ctx context.Context, l logr.Logger) (*mstorage.Storage, error) {
	minioOnce.Do(func() {
		// find the secret "minio" in the "minio" namespace
		secret := &corev1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: "minio", Name: "minio"}, secret); err != nil {
			l.Error(err, "secret not found")
			return
		}
		// l.Info("minio secret found", "secret.Data", secret.Data)
		user := string(secret.Data["root-user"])
		// l.Info("base64", "user", user)
		password := string(secret.Data["root-password"])
		// l.Info("base64", "password", password)

		viper.Set("minio-access-key", user)
		viper.Set("minio-secret-key", password)

		if _, err := mstorage.NewStorage(logrus.New()); err != nil {
			l.Error(err, "unable to create minio client")
			return
		}
		minioInitialized = true
	})

	if !minioInitialized {
		return nil, fmt.Errorf("minio client not initialized")
	}

	return mstorage.Instance(), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&integrityv1.Snapshot{}).
		Complete(r)
}

// TODO: move to pkg/minio
//
// BuildObjectName returns the object name for the given @namespace and @image.
//
// An @image has the following format: imageName:imageTag
// Returns: namespace/imageName/imageTag
func buildObjectName(namespace, image string) string {
	imageInfo := strings.Split(image, ":")
	return namespace + "/" + imageInfo[0] + "/" + imageInfo[1]
}

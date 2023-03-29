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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	integrityv1 "integrity/snapshot/api/v1"

	mstorage "github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
)

// SnapshotReconciler reconciles a Snapshot object
type SnapshotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	// Minio  *minio.Client
}

//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=secrets/status,verbs=get

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
	// l := log.FromContext(ctx)
	var snapshot integrityv1.Snapshot
	if err := r.Get(ctx, req.NamespacedName, &snapshot); err != nil {
		// r.Log.Error(err, "unable to fetch snapshot")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.Log.V(1).Info("snapshot found", "image", snapshot.Spec.Image)

	updated, err := r.updateSnapshotFinalizer(ctx, &snapshot)
	if err != nil {
		r.Log.Error(err, "unable to update snapshot finalizer")
		return ctrl.Result{}, err
	}
	if updated {
		// object has changed and will be uploaded to MinIO during next call to
		// reconcile to prevent uploading twice
		return ctrl.Result{}, nil
	}

	// TODO: //controllerutil.AddFinalizer(cronJob, finalizerName)

	// check that deletion process is started
	if snapshot.DeletionTimestamp != nil {
		r.Log.V(1).Info("snapshot is being deleted", "snapshot", snapshot.Name)
		if err = r.deleteSnapshot(ctx, &snapshot); err != nil {
			r.Log.Error(err, "unable to delete snapshot", "snapshot", snapshot.Name)
			return ctrl.Result{}, err
		}
		r.Log.V(1).Info("snapshot has been deleted", "snapshot", snapshot.Name)
		return ctrl.Result{}, nil
	}

	// if !snapshot.Status.IsUploaded {
	err = r.uploadSnapshot(ctx, snapshot, req)
	if err != nil {
		r.Log.Error(err, "unable to upload snapshot")
		return ctrl.Result{}, err
	}
	// }
	r.Log.V(1).Info("all snapshots uploaded")

	return ctrl.Result{}, nil
}

func (r *SnapshotReconciler) deleteSnapshot(ctx context.Context, obj *integrityv1.Snapshot) error {
	// removing object from the MinIO storage
	if err := r.removeSnapshot(ctx, obj); err != nil {
		r.Log.Error(err, "unable to delete object", "snapshot", obj.Name)
		return err
	}

	r.Log.V(1).Info("removing finalizer", "snapshot", obj.Name)
	r.removeFinalizer(obj)
	if err := r.Update(ctx, obj); err != nil {
		r.Log.Error(err, "unable to update/remove finalizer", "snapshot", obj.Name)
		return err
	}
	return nil
}

func (r *SnapshotReconciler) removeFinalizer(obj *integrityv1.Snapshot) {
	for i, v := range obj.ObjectMeta.Finalizers {
		if v == finalizerName {
			obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers[:i], obj.ObjectMeta.Finalizers[i+1:]...)
			break
		}
	}
}

const finalizerName = "controller.snapshot/finalizer"

// returns true if object was updated and error if any error occured
func (r *SnapshotReconciler) updateSnapshotFinalizer(
	ctx context.Context,
	obj *integrityv1.Snapshot,
) (bool, error) {
	var isFound bool
	for _, v := range obj.ObjectMeta.Finalizers {
		// if strings.Contains(v, finalizerName) {
		r.Log.V(1).Info("finalizer found", "finalizer", v)
		if v == finalizerName {
			isFound = true
			break
		}
	}

	if !isFound {
		r.Log.V(1).Info("updating finalizer", "snapshot", obj.Name)
		obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers, finalizerName)
		err := r.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "unable to update finalizer")
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *SnapshotReconciler) uploadSnapshot(
	ctx context.Context,
	snapshot integrityv1.Snapshot,
	req reconcile.Request,
) error {
	ms, err := r.minIOStorage(ctx, r.Log)
	if err != nil {
		r.Log.Error(err, "unable to get MinIO client")
		return err
	}

	// TODO: move to MinIO
	imageInfo := strings.Split(snapshot.Spec.Image, ":")
	objName := req.NamespacedName.Namespace + "/" + imageInfo[0] + "/" + imageInfo[1]

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// TODO: mstorage.DefaultBucketName "integrity"
	// TODO: mstorage.BuildObjectName(req.NamespacedName.Namespace, snapshot.Spec.Image),
	if err = ms.Save(ctx, "integrity",
		objName,
		[]byte(snapshot.Spec.Base64Hashes),
	); err != nil {
		return err
	}
	r.Log.Info("snapshot saved", "image", snapshot.Spec.Image)
	r.Log.Info("") // TODO: remove this

	// snapshot.Status.IsUploaded = true
	// if err := r.Status().Update(ctx, &snapshot); err != nil {
	// 	r.Log.Error(err, "unable to update snapshot status")
	// 	return err
	// }
	// r.Log.V(1).Info("snapshot status updated", "snapshot.Status", snapshot.Status)

	return nil
}

// removes data related to @obj from the MinIO storage
func (r *SnapshotReconciler) removeSnapshot(ctx context.Context, obj *integrityv1.Snapshot) error {
	// imageInfo := strings.Split(obj.Spec.Image, ":")
	// objName := obj.Namespace + "/" + imageInfo[0] + "/" + imageInfo[1]
	// ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()

	// ms, err := r.minIOStorage(ctx, r.Log)
	// if err != nil {
	// 	r.Log.Error(err, "unable to get MinIO client")
	// 	return err
	// }
	// // TODO: "integrity"
	// if err = ms.Remove(ctx, "integrity", objName); err != nil {
	// 	r.Log.Error(err, "unable to remove object from MinIO storage", "snapshot", obj.Name)
	// 	return err
	// }
	return nil
}

var (
	minioOnce        sync.Once
	minioInitialized bool
)

func (r *SnapshotReconciler) minIOStorage(
	ctx context.Context,
	l logr.Logger,
) (*mstorage.Storage, error) {
	minioOnce.Do(func() {
		// find the secret "minio" in the "minio" namespace
		secret := &corev1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: "minio", Name: "minio"}, secret); err != nil {
			r.Log.Error(err, "secret not found")
			return
		}
		user := string(secret.Data["root-user"])
		password := string(secret.Data["root-password"])
		viper.Set("minio-access-key", user)
		viper.Set("minio-secret-key", password)

		if _, err := mstorage.NewStorage(logrus.New()); err != nil {
			r.Log.Error(err, "unable to create minio client")
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

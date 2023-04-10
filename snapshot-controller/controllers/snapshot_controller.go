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
	"crypto/md5"
	"fmt"
	integrityv1 "integrity/snapshot/api/v1"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mstorage "github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
)

// SnapshotReconciler reconciles a Snapshot object
type SnapshotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

const finalizerName = "controller.snapshot/finalizer"

//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=integrity.snapshot,resources=snapshots/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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
	updateStatus := func(ctx context.Context, obj *integrityv1.Snapshot) error {
		if err := r.Status().Update(ctx, obj); err != nil {
			r.Log.Error(err, "unable to update snapshot status", "snapshot", obj.Name)
			return err
		}
		r.Log.V(1).Info("snapshot status has been updated", "snapshot", obj.Name,
			"IsUploaded", obj.Status.IsUploaded)
		return nil
	}

	ms, err := r.minIOStorage(ctx)
	if err != nil {
		r.Log.Error(err, "unable to get MinIO client")
		return ctrl.Result{}, err
	}

	var snapshot integrityv1.Snapshot
	if err = r.Get(ctx, req.NamespacedName, &snapshot); err != nil {
		if !errors.IsNotFound(err) {
			r.Log.Error(err, "unable to fetch snapshot")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.Log.V(1).Info("snapshot found",
		"snapshot", snapshot.Name,
		"image", snapshot.Spec.Image,
		"IsUploaded", snapshot.Status.IsUploaded)

	// check that deletion process is started
	if snapshot.DeletionTimestamp != nil {
		r.Log.V(1).Info("snapshot is being deleted", "snapshot", snapshot.Name)
		if err := r.deleteSnapshot(ctx, ms, &snapshot); err != nil {
			r.Log.Error(err, "unable to delete snapshot", "snapshot", snapshot.Name)
			return ctrl.Result{}, err
		}
		r.Log.V(1).Info("snapshot has been deleted", "snapshot", snapshot.Name)
		return ctrl.Result{}, nil
	}

	// upload if needed
	controlHash := md5hash(snapshot.Spec.Base64Hashes)
	if controlHash != snapshot.Status.ControlHash || !snapshot.Status.IsUploaded {
		if err := r.uploadSnapshot(ctx, ms, snapshot, req); err != nil {
			r.Log.Error(err, "unable to upload snapshot")
			snapshot.Status.IsUploaded = false
			r.Log.V(1).Info("not uploaded, changing status to false")
			_ = updateStatus(ctx, &snapshot)
			return ctrl.Result{}, nil
		}
		snapshot.Status.ControlHash, snapshot.Status.IsUploaded = controlHash, true
		err = updateStatus(ctx, &snapshot)
		if err != nil {
			r.Log.Error(err, "unable to update snapshot status", "snapshot", snapshot.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// ..deletion process for the @obj
func (r *SnapshotReconciler) deleteSnapshot(
	ctx context.Context,
	ms *mstorage.Storage,
	obj *integrityv1.Snapshot,
) error {
	if err := r.removeSnapshot(ctx, ms, obj); err != nil {
		r.Log.Error(err, "unable to delete object", "snapshot", obj.Name)
		return err
	}

	r.Log.V(1).Info("removing finalizer", "snapshot", obj.Name)
	controllerutil.RemoveFinalizer(obj, finalizerName)
	if err := r.Update(ctx, obj); err != nil {
		r.Log.Error(err, "unable to update/remove finalizer", "snapshot", obj.Name)
		return err
	}
	return nil
}

// ..uploads data related to the @obj to the MinIO storage
func (r *SnapshotReconciler) uploadSnapshot(
	ctx context.Context,
	ms *mstorage.Storage,
	o integrityv1.Snapshot,
	req reconcile.Request,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectName := mstorage.BuildObjectName(req.NamespacedName.Namespace, o.Spec.Image, o.Spec.Algorithm)
	if err := ms.Save(ctx, mstorage.DefaultBucketName, objectName, []byte(o.Spec.Base64Hashes)); err != nil {
		return err
	}
	r.Log.Info("snapshot uploaded", "objectName", objectName, "IsUploaded", o.Status.IsUploaded)
	return nil
}

// ..removes data related to @obj from the MinIO storage
func (r *SnapshotReconciler) removeSnapshot(
	ctx context.Context,
	ms *mstorage.Storage,
	obj *integrityv1.Snapshot,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	objName := mstorage.BuildObjectName(obj.Namespace, obj.Spec.Image, obj.Spec.Algorithm)
	if err := ms.Remove(ctx, mstorage.DefaultBucketName, objName); err != nil {
		r.Log.Error(err, "unable to remove object from MinIO storage", "snapshot", obj.Name)
		return err
	}
	return nil
}

var (
	minioOnce        sync.Once
	minioInitialized bool
)

// ..initializes the MinIO storage and returns it instance
func (r *SnapshotReconciler) minIOStorage(ctx context.Context) (*mstorage.Storage, error) {
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

// ..calculates and returns md5 hash of the string
func md5hash(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// SetupWithManager sets up the controller with the Manager.
func (r *SnapshotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&integrityv1.Snapshot{}).
		Complete(r)
}

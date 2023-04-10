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
	integrityv1 "integrity/snapshot/api/v1"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	//+kubebuilder:scaffold:imports

	mstorage "github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	k8sClient  client.Client
	testEnv    *envtest.Environment
	k8sManager manager.Manager
	k8sLogger  logr.Logger
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var portForwardProcessPid int

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	useExistingCluster := true
	testEnv = &envtest.Environment{
		UseExistingCluster: &useExistingCluster,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = integrityv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	// get the manager..
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())
	k8sClient = k8sManager.GetClient()
	k8sLogger = ctrl.Log.WithName("controllers").WithName("snapshot-test")

	// ..and start the snapshot controller
	err = (&SnapshotReconciler{
		Client: k8sClient,
		Scheme: k8sManager.GetScheme(),
		Log:    k8sLogger,
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	By("port-forwarding for the MinIO service")
	cmd := exec.Command("kubectl", "port-forward", "svc/minio", "-n", "minio", "9000:9000")
	Expect(cmd.Start()).To(Succeed())
	println("port-forward process PID:", cmd.Process.Pid)
	portForwardProcessPid = cmd.Process.Pid
	viper.Set("minio-host", "127.0.0.1:9000")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	go func() {
		defer GinkgoRecover()
		defer cancel()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

	if portForwardProcessPid != 0 {
		By("stop port-forwarding MinIO service")
		cmd := exec.Command("kill", "-9", strconv.Itoa(portForwardProcessPid))
		Expect(cmd.Run()).To(Succeed())
	}
})

var _ = Describe("SnapshotController", func() {
	var (
		toCreate  *integrityv1.Snapshot
		ctx       context.Context
		r         *SnapshotReconciler
		fetched   *integrityv1.Snapshot
		objectKey types.NamespacedName
		objName   string
	)

	BeforeEach(func() {
		toCreate = &integrityv1.Snapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-test",
				Namespace: "default",
				Finalizers: []string{
					finalizerName,
				},
			},
			Spec: integrityv1.SnapshotSpec{
				Image:        "imageName:imageTag",
				Base64Hashes: "aGFzaGVzCg==",
				Algorithm:    "md5",
			},
		}

		fetched = new(integrityv1.Snapshot)
		objectKey = types.NamespacedName{
			Name:      toCreate.Name,
			Namespace: toCreate.Namespace,
		}
		r = &SnapshotReconciler{
			Client: k8sClient,
			Log:    k8sLogger,
		}

		objName = mstorage.BuildObjectName(
			toCreate.Namespace,
			toCreate.Spec.Image,
			toCreate.Spec.Algorithm,
		)
		ctx = context.Background()
	})

	It("testing snapshot CR, controller & interaction w/ Minio", func() {
		By("ensure the object is not created")
		_ = k8sClient.Delete(ctx, toCreate)
		// let's wait a little while the controller Reconcile the new object
		time.Sleep(400 * time.Millisecond)

		By("create test snapshot CR")
		Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())
		time.Sleep(200 * time.Millisecond)

		By("verify test snapshot CRD on the cluster")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, objectKey, fetched)
			return err == nil
		}).Should(BeTrue())

		fmt.Printf("-- debug fetched.Status: %+v\n", fetched.Status)

		Expect(toCreate.Name).To(Equal(fetched.Name))
		Expect(fetched.Status.IsUploaded).To(BeTrue())

		_, err := r.minIOStorage(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(mstorage.Instance()).NotTo(BeNil())

		By("load and verify the MinIO object")
		bs, err := mstorage.Instance().Load(ctx, mstorage.DefaultBucketName, objName)
		Expect(err).NotTo(HaveOccurred())
		Expect(bs).To(HaveLen(12))
		Expect(string(bs)).To(Equal(toCreate.Spec.Base64Hashes))

		By("delete test snapshot CRD")
		Expect(k8sClient.Delete(ctx, toCreate)).To(Succeed())
		time.Sleep(200 * time.Millisecond)

		By("try to get the deleted before CRD")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, objectKey, fetched)
			return err == nil
		}).Should(BeFalse())

		_, err = mstorage.Instance().Load(ctx, mstorage.DefaultBucketName, objName)
		Expect(err).ShouldNot(Succeed())
	})
})

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

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg     *rest.Config
	testEnv *envtest.Environment
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

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
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("e2e test", func() {

	It("testing integrity", func() {
		By("new file added")
		k8sClient, err := kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred(), "failed to get client")
		podName, podNamespace, err := GetPodName(k8sClient)
		fmt.Println("pod spec", podName, podNamespace)
		Expect(err).ToNot(HaveOccurred())
		cmd := exec.Command("kubectl", "-n", podNamespace, "exec", "-it", podName, "--", "touch", "/usr/bin/newfile")
		Expect(cmd.Start()).To(Succeed())

		// give a time to check files
		time.Sleep(2 * time.Second)
		getLogs := exec.Command("kubectl", "-n", podNamespace, "logs", "svc/rsyslog")
		out, err := getLogs.Output()
		Expect(err).ToNot(HaveOccurred(), "failed to get logs")
		msg := GetLastMsg(out)
		fmt.Println(msg)
		expectRegexp := "\\d*-\\d*-\\d*T\\d*:\\d*:\\d*\\+\\d*:\\d*\\sapp-nginx-integrity\\.default\\sintegrity-monitor\\[\\d*\\]:\\stime=\\w*\\s*\\d*\\s\\d*:\\d*:\\d*\\sevent-type=\\d*\\sservice=nginx\\spod=app-nginx-integrity-\\w*-\\w*\\simage=nginx:\\d*.\\d*.\\d*\\snamespace=default\\scluster=local\\smessage=Restart pod app-nginx-integrity-\\w*-\\w*\\sfile=usr\\/bin\\/newfile\\sreason=new\\sfile\\sfound"
		Expect(msg).To(MatchRegexp(expectRegexp))

		By("file deleted")
		// waiting pod restart
		time.Sleep(5 * time.Second)
		k8sClient, err = kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred(), "failed to get client")
		podName, podNamespace, err = GetPodName(k8sClient)
		Expect(err).ToNot(HaveOccurred())
		cmd = exec.Command("kubectl", "-n", podNamespace, "exec", "-it", podName, "--", "rm", "-f", "/usr/bin/tr")
		Expect(cmd.Start()).To(Succeed())

		// give a time to check files
		time.Sleep(2 * time.Second)
		getLogs = exec.Command("kubectl", "-n", podNamespace, "logs", "svc/rsyslog")
		out, err = getLogs.Output()
		Expect(err).ToNot(HaveOccurred(), "failed to get logs")
		msg = GetLastMsg(out)
		fmt.Println(msg)
		expectRegexp = "reason=file deleted"
		Expect(msg).To(MatchRegexp(expectRegexp))

		By("file changed")
		// waiting pod restart
		time.Sleep(5 * time.Second)
		k8sClient, err = kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred(), "failed to get client")
		podName, podNamespace, err = GetPodName(k8sClient)
		Expect(err).ToNot(HaveOccurred())
		cmd = exec.Command("kubectl", "-n", podNamespace, "exec", "-it", podName, "--", "cp", "/usr/bin/cut", "/usr/bin/tr")
		Expect(cmd.Start()).To(Succeed())

		// give a time to check files
		time.Sleep(2 * time.Second)
		getLogs = exec.Command("kubectl", "-n", podNamespace, "logs", "svc/rsyslog")
		out, err = getLogs.Output()
		Expect(err).ToNot(HaveOccurred(), "failed to get logs")
		msg = GetLastMsg(out)
		fmt.Println(msg)
		expectRegexp = "reason=file content mismatch"
		Expect(msg).To(MatchRegexp(expectRegexp))
	})
})

func GetLastMsg(in []byte) string {
	r := bufio.NewReader(bytes.NewBuffer(in))
	sc := bufio.NewScanner(r)
	msg := ""
	for sc.Scan() {
		msg = sc.Text()
	}

	return msg
}

func GetPodName(k8sClient *kubernetes.Clientset) (string, string, error) {
	pods, err := k8sClient.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=nginx-app",
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to get pods %w", err)
	}
	if len(pods.Items) == 0 {
		return "", "", fmt.Errorf("%s", "cannot find pod")
	}

	return pods.Items[0].Name, pods.Items[0].Namespace, nil
}

/*
Copyright 2024.

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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
)

const (
	testNamespace = "e2e-test"
	poolPrefix    = "e2e-pool-"
	appPrefix     = "e2e-app-"
	maxWait       = 5 * time.Minute
	pollInterval  = 5 * time.Second
)

var (
	existedLBID        = os.Getenv("E2E_EXISTED_LB_ID")
	lbRegion           = os.Getenv("E2E_LB_REGION")
	vpcID              = os.Getenv("E2E_VPC_ID")
	skipExistedLBTests = os.Getenv("E2E_SKIP_EXISTED_LB_TESTS") == "true"
	testImage          = os.Getenv("E2E_TEST_IMAGE")
)

var k8sClient client.Client

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	Expect(networkingv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	kubeconfig := os.Getenv("KUBECONFIG")
	Expect(kubeconfig).NotTo(BeEmpty(), "KUBECONFIG 环境变量必须设置")
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// 创建测试命名空间
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	err = k8sClient.Create(context.Background(), ns)
	if !apierrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}

	// 等待 controller pod 就绪
	Eventually(func() bool {
		pods := &corev1.PodList{}
		err := k8sClient.List(context.Background(), pods,
			client.InNamespace("kube-system"),
			client.MatchingLabels{"app.kubernetes.io/name": "tke-extend-network-controller"})
		if err != nil || len(pods.Items) == 0 {
			return false
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false
			}
		}
		return true
	}, maxWait, pollInterval).Should(BeTrue(), "tke-extend-network-controller 应该已部署并运行")
})

var _ = AfterSuite(func() {
	_ = k8sClient.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}})
	pools := &networkingv1alpha1.CLBPortPoolList{}
	if err := k8sClient.List(context.Background(), pools); err == nil {
		for _, pool := range pools.Items {
			if strings.HasPrefix(pool.Name, poolPrefix) {
				_ = k8sClient.Delete(context.Background(), &pool)
			}
		}
	}
})

// ============= 辅助函数 =============

func getDefaultRegion() string {
	if lbRegion != "" {
		return lbRegion
	}
	return "ap-shanghai"
}

func getImage() string {
	if testImage == "" {
		return "nginx:alpine"
	}
	return testImage
}

func autoCreateLBSpec(startPort uint16) networkingv1alpha1.CLBPortPoolSpec {
	return networkingv1alpha1.CLBPortPoolSpec{
		StartPort: startPort,
		Region:    ptr(getDefaultRegion()),
		AutoCreate: &networkingv1alpha1.AutoCreateConfig{
			Enabled: true,
			Parameters: &networkingv1alpha1.CreateLBParameters{
				VpcId: ptr(vpcID),
			},
		},
	}
}

func createPool(ctx context.Context, name string, spec networkingv1alpha1.CLBPortPoolSpec) {
	pool := &networkingv1alpha1.CLBPortPool{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       spec,
	}
	Expect(k8sClient.Create(ctx, pool)).To(Succeed())
}

func deletePool(ctx context.Context, name string) {
	pool := &networkingv1alpha1.CLBPortPool{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, pool); err != nil {
		return
	}
	_ = k8sClient.Delete(ctx, pool)
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, pool)
		return apierrors.IsNotFound(err)
	}, maxWait, pollInterval).Should(BeTrue(), fmt.Sprintf("CLBPortPool %s 应该被删除", name))
}

func waitPoolActive(ctx context.Context, name string) *networkingv1alpha1.CLBPortPool {
	var pool *networkingv1alpha1.CLBPortPool
	Eventually(func() string {
		pool = &networkingv1alpha1.CLBPortPool{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, pool); err != nil {
			return ""
		}
		return string(pool.Status.State)
	}, maxWait, pollInterval).Should(Equal("Active"), fmt.Sprintf("CLBPortPool %s 应该变为 Active", name))
	return pool
}

func createDep(ctx context.Context, name string, annotations map[string]string, replicas int32, ports ...int32) *appsv1.Deployment {
	containerPorts := make([]corev1.ContainerPort, len(ports))
	for i, p := range ports {
		containerPorts[i] = corev1.ContainerPort{ContainerPort: p}
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: getImage(),
						Ports: containerPorts,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					}},
				},
			},
		},
	}
	Expect(k8sClient.Create(ctx, dep)).To(Succeed())
	return dep
}

func mappingAnnotations(port int32, protocol, poolStr string, extra ...string) map[string]string {
	val := fmt.Sprintf("%d %s %s", port, protocol, poolStr)
	if len(extra) > 0 {
		val += " " + strings.Join(extra, " ")
	}
	return map[string]string{
		constant.EnableCLBPortMappingsKey: "true",
		constant.CLBPortMappingsKey:       val,
	}
}

func deleteDep(ctx context.Context, name string) {
	_ = k8sClient.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}})
	Eventually(func() bool {
		pods := &corev1.PodList{}
		_ = k8sClient.List(ctx, pods, client.InNamespace(testNamespace), client.MatchingLabels{"app": name})
		return len(pods.Items) == 0
	}, maxWait, pollInterval).Should(BeTrue())
}

// waitMappingReady 等待 appLabel 对应的单副本 Pod 获得 Ready 映射结果。
// 仅适用于单副本场景；多副本请用 waitAllMappingsReady。
func waitMappingReady(ctx context.Context, appLabel string) []PortMappingResult {
	var mappings []PortMappingResult
	Eventually(func() bool {
		pods := &corev1.PodList{}
		err := k8sClient.List(ctx, pods, client.InNamespace(testNamespace), client.MatchingLabels{"app": appLabel})
		if err != nil || len(pods.Items) == 0 {
			return false
		}
		pod := pods.Items[0]
		if pod.Annotations[constant.CLBPortMappingStatuslKey] != "Ready" {
			return false
		}
		result := pod.Annotations[constant.CLBPortMappingResultKey]
		if result == "" {
			return false
		}
		Expect(json.Unmarshal([]byte(result), &mappings)).To(Succeed())
		return true
	}, maxWait, pollInterval).Should(BeTrue(),
		fmt.Sprintf("app=%s 的 Pod 应该有 Ready 的 CLB 端口映射结果", appLabel))
	return mappings
}

func waitAllMappingsReady(ctx context.Context, appLabel string, count int) [][]PortMappingResult {
	var results [][]PortMappingResult
	Eventually(func() int {
		pods := &corev1.PodList{}
		_ = k8sClient.List(ctx, pods, client.InNamespace(testNamespace), client.MatchingLabels{"app": appLabel})
		ready := 0
		results = make([][]PortMappingResult, 0, count)
		for _, pod := range pods.Items {
			if pod.Annotations[constant.CLBPortMappingStatuslKey] == "Ready" {
				result := pod.Annotations[constant.CLBPortMappingResultKey]
				if result != "" {
					var m []PortMappingResult
					if json.Unmarshal([]byte(result), &m) == nil {
						results = append(results, m)
						ready++
					}
				}
			}
		}
		return ready
	}, maxWait, pollInterval).Should(Equal(count), fmt.Sprintf("app=%s 的所有 Pod 应该有 Ready 的映射结果", appLabel))
	return results
}

func getPodBindingState(ctx context.Context, podName string) string {
	binding := &networkingv1alpha1.CLBPodBinding{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: podName, Namespace: testNamespace}, binding); err != nil {
		return ""
	}
	return string(binding.Status.State)
}

func getFirstPodName(ctx context.Context, appLabel string) string {
	pods := &corev1.PodList{}
	Expect(k8sClient.List(ctx, pods, client.InNamespace(testNamespace), client.MatchingLabels{"app": appLabel})).To(Succeed())
	Expect(pods.Items).NotTo(BeEmpty())
	return pods.Items[0].Name
}

type PortMappingResult struct {
	Port                uint16   `json:"port"`
	Protocol            string   `json:"protocol"`
	Pool                string   `json:"pool"`
	Region              string   `json:"region"`
	LoadbalancerId      string   `json:"loadbalancerId"`
	LoadbalancerPort    uint16   `json:"loadbalancerPort"`
	LoadbalancerEndPort *uint16  `json:"loadbalancerEndPort,omitempty"`
	ListenerId          string   `json:"listenerId"`
	Hostname            *string  `json:"hostname,omitempty"`
	Ips                 []string `json:"ips,omitempty"`
	Address             string   `json:"address,omitempty"`
}

func findMapping(mappings []PortMappingResult, protocol string) *PortMappingResult {
	for i := range mappings {
		if mappings[i].Protocol == protocol {
			return &mappings[i]
		}
	}
	return nil
}

func findMappingByPort(mappings []PortMappingResult, port uint16) *PortMappingResult {
	for i := range mappings {
		if mappings[i].Port == port {
			return &mappings[i]
		}
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

// ============= 测试用例 =============

var _ = Describe("tke-extend-network-controller e2e", func() {
	ctx := context.Background()

	// 1. CLBPortPool 基本功能
	Describe("CLBPortPool 基本功能", func() {
		It("应能创建带自动创建 CLB 的端口池，分配端口时自动创建 CLB", func() {
			poolName := poolPrefix + "auto-create"
			createPool(ctx, poolName, autoCreateLBSpec(30000))
			defer deletePool(ctx, poolName)

			pool := waitPoolActive(ctx, poolName)
			Expect(pool.Status.State).To(Equal(networkingv1alpha1.CLBPortPoolStateActive))

			depName := appPrefix + "pool-auto"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
			Expect(mappings[0].LoadbalancerId).NotTo(BeEmpty())

			Eventually(func() int {
				updated := &networkingv1alpha1.CLBPortPool{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: poolName}, updated)
				return len(updated.Status.LoadbalancerStatuses)
			}, maxWait, pollInterval).Should(BeNumerically(">", 0))
		})

		It("应能创建使用已有 CLB 的端口池并变为 Active", func() {
			if skipExistedLBTests || existedLBID == "" {
				Skip("跳过已有 CLB 测试")
			}
			poolName := poolPrefix + "existed-lb"
			spec := networkingv1alpha1.CLBPortPoolSpec{
				StartPort:               30000,
				Region:                  ptr(getDefaultRegion()),
				ExsistedLoadBalancerIDs: []string{existedLBID},
			}
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)

			pool := waitPoolActive(ctx, poolName)
			Expect(pool.Status.LoadbalancerStatuses).NotTo(BeEmpty())
			Expect(pool.Status.LoadbalancerStatuses[0].LoadbalancerID).To(Equal(existedLBID))
		})
	})

	// 2. TCP 端口映射
	Describe("Pod TCP 端口映射", func() {
		It("应为 TCP Pod 分配 CLB 端口映射并写入注解", func() {
			poolName := poolPrefix + "tcp"
			createPool(ctx, poolName, autoCreateLBSpec(30100))
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "tcp"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
			Expect(mappings[0].Protocol).To(Equal("TCP"))
			Expect(mappings[0].Port).To(Equal(uint16(80)))
			Expect(mappings[0].Pool).To(Equal(poolName))
			Expect(mappings[0].LoadbalancerId).NotTo(BeEmpty())
			Expect(mappings[0].ListenerId).NotTo(BeEmpty())
			Expect(mappings[0].LoadbalancerPort).To(BeNumerically(">=", 30100))
		})
	})

	// 3. UDP 端口映射
	Describe("Pod UDP 端口映射", func() {
		It("应为 UDP Pod 分配 CLB 端口映射", func() {
			poolName := poolPrefix + "udp"
			createPool(ctx, poolName, autoCreateLBSpec(30200))
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "udp"
			createDep(ctx, depName, mappingAnnotations(8080, "UDP", poolName), 1, 8080)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
			Expect(mappings[0].Protocol).To(Equal("UDP"))
			Expect(mappings[0].Port).To(Equal(uint16(8080)))
		})
	})

	// 4. TCPUDP 协议
	Describe("Pod TCPUDP 端口映射", func() {
		It("应为 TCPUDP Pod 同时分配 TCP 和 UDP 映射，且端口号相同", func() {
			poolName := poolPrefix + "tcpudp"
			createPool(ctx, poolName, autoCreateLBSpec(30300))
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "tcpudp"
			createDep(ctx, depName, mappingAnnotations(9000, "TCPUDP", poolName), 1, 9000)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).To(HaveLen(2), "TCPUDP 应该产生 2 个映射（TCP + UDP）")
			tcp := findMapping(mappings, "TCP")
			udp := findMapping(mappings, "UDP")
			Expect(tcp).NotTo(BeNil())
			Expect(udp).NotTo(BeNil())
			Expect(tcp.LoadbalancerPort).To(Equal(udp.LoadbalancerPort), "TCP 和 UDP 应使用相同 CLB 端口号")
			Expect(tcp.LoadbalancerId).To(Equal(udp.LoadbalancerId), "TCP 和 UDP 应使用相同 CLB 实例")
		})
	})

	// 5. 多端口池映射
	Describe("多端口池映射", func() {
		It("应从多个端口池分别为 Pod 分配映射", func() {
			pool1 := poolPrefix + "multi-1"
			pool2 := poolPrefix + "multi-2"
			createPool(ctx, pool1, autoCreateLBSpec(30400))
			defer deletePool(ctx, pool1)
			createPool(ctx, pool2, autoCreateLBSpec(30500))
			defer deletePool(ctx, pool2)
			waitPoolActive(ctx, pool1)
			waitPoolActive(ctx, pool2)

			depName := appPrefix + "multi-pool"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", pool1+","+pool2), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).To(HaveLen(2), "两个端口池应该各产生一个映射")
			pools := []string{mappings[0].Pool, mappings[1].Pool}
			Expect(pools).To(ContainElement(pool1))
			Expect(pools).To(ContainElement(pool2))
		})
	})

	// 6. useSamePortAcrossPools
	Describe("useSamePortAcrossPools", func() {
		It("跨端口池应分配相同的 CLB 端口号", func() {
			pool1 := poolPrefix + "same-port-1"
			pool2 := poolPrefix + "same-port-2"
			createPool(ctx, pool1, autoCreateLBSpec(30600))
			defer deletePool(ctx, pool1)
			createPool(ctx, pool2, autoCreateLBSpec(30700))
			defer deletePool(ctx, pool2)
			waitPoolActive(ctx, pool1)
			waitPoolActive(ctx, pool2)

			depName := appPrefix + "same-port"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", pool1+","+pool2, "useSamePortAcrossPools"), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).To(HaveLen(2))
			Expect(mappings[0].LoadbalancerPort).To(Equal(mappings[1].LoadbalancerPort),
				"useSamePortAcrossPools 应确保两个端口池分配相同 CLB 端口号")
		})
	})

	// 7. 注解移除清理
	Describe("注解移除清理", func() {
		It("移除 enable 注解后 CLBPodBinding 应被删除", func() {
			poolName := poolPrefix + "cleanup"
			createPool(ctx, poolName, autoCreateLBSpec(30800))
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "cleanup"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			waitMappingReady(ctx, depName)

			// 获取 Pod 名称
			podName := getFirstPodName(ctx, depName)

			// 移除注解
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: depName, Namespace: testNamespace}, dep)).To(Succeed())
			delete(dep.Spec.Template.Annotations, constant.EnableCLBPortMappingsKey)
			delete(dep.Spec.Template.Annotations, constant.CLBPortMappingsKey)
			Expect(k8sClient.Update(ctx, dep)).To(Succeed())

			// 等待 CLBPodBinding 被删除
			Eventually(func() bool {
				binding := &networkingv1alpha1.CLBPodBinding{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: podName, Namespace: testNamespace}, binding)
				return apierrors.IsNotFound(err)
			}, maxWait, pollInterval).Should(BeTrue(), "CLBPodBinding 应该被删除")

			// 等待新 Pod 注解被清除
			Eventually(func() bool {
				newPods := &corev1.PodList{}
				_ = k8sClient.List(ctx, newPods, client.InNamespace(testNamespace), client.MatchingLabels{"app": depName})
				if len(newPods.Items) == 0 {
					return false
				}
				pod := newPods.Items[0]
				_, hasResult := pod.Annotations[constant.CLBPortMappingResultKey]
				_, hasStatus := pod.Annotations[constant.CLBPortMappingStatuslKey]
				return !hasResult && !hasStatus
			}, maxWait, pollInterval).Should(BeTrue(), "Pod 注解应该被清除")
		})
	})

	// 8. 禁用映射
	Describe("禁用映射", func() {
		It("设置 enable=false 后 CLBPodBinding 应处于 Disabled 状态", func() {
			poolName := poolPrefix + "disable"
			createPool(ctx, poolName, autoCreateLBSpec(30900))
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "disable"
			ann := mappingAnnotations(80, "TCP", poolName)
			ann[constant.EnableCLBPortMappingsKey] = "false"
			createDep(ctx, depName, ann, 1, 80)
			defer deleteDep(ctx, depName)

			// 等待 Pod 就绪
			Eventually(func() bool {
				pods := &corev1.PodList{}
				_ = k8sClient.List(ctx, pods, client.InNamespace(testNamespace), client.MatchingLabels{"app": depName})
				return len(pods.Items) > 0
			}, maxWait, pollInterval).Should(BeTrue())

			podName := getFirstPodName(ctx, depName)

			Eventually(func() string {
				return getPodBindingState(ctx, podName)
			}, maxWait, pollInterval).Should(Equal("Disabled"), "CLBPodBinding 状态应该为 Disabled")
		})
	})

	// 9. 多端口映射
	Describe("多端口映射", func() {
		It("应为 Pod 的多个端口分别分配映射", func() {
			poolName := poolPrefix + "multi-port"
			createPool(ctx, poolName, autoCreateLBSpec(31000))
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "multi-port"
			ann := map[string]string{
				constant.EnableCLBPortMappingsKey: "true",
				constant.CLBPortMappingsKey:       fmt.Sprintf("80 TCP %s\n9000 UDP %s", poolName, poolName),
			}
			createDep(ctx, depName, ann, 1, 80, 9000)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).To(HaveLen(2), "两个端口应该各产生一个映射")
			Expect(findMappingByPort(mappings, 80)).NotTo(BeNil())
			Expect(findMappingByPort(mappings, 9000)).NotTo(BeNil())
		})
	})

	// 10. LB 分配策略
	Describe("LB 分配策略", func() {
		It("应支持 Uniform 分配策略", func() {
			poolName := poolPrefix + "policy-uniform"
			policy := "Uniform"
			spec := autoCreateLBSpec(31100)
			spec.LbPolicy = &policy
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "policy-uniform"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
		})
	})

	// 11. 端口池不存在
	Describe("端口池不存在", func() {
		It("CLBPodBinding 应报告 PortPoolNotFound 状态", func() {
			depName := appPrefix + "no-pool"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", "nonexistent-pool"), 1, 80)
			defer deleteDep(ctx, depName)

			Eventually(func() bool {
				pods := &corev1.PodList{}
				_ = k8sClient.List(ctx, pods, client.InNamespace(testNamespace), client.MatchingLabels{"app": depName})
				return len(pods.Items) > 0
			}, maxWait, pollInterval).Should(BeTrue())

			podName := getFirstPodName(ctx, depName)
			Eventually(func() string {
				return getPodBindingState(ctx, podName)
			}, maxWait, pollInterval).Should(Equal("PortPoolNotFound"))
		})
	})

	// 12. 端口池自动扩容
	Describe("端口池自动扩容", func() {
		It("端口不足时应自动创建新 CLB", func() {
			poolName := poolPrefix + "scaleup"
			spec := autoCreateLBSpec(31200)
			// 限制端口范围，5 个 Pod x TCPUDP(2 listener) = 10 个监听器
			// endPort=31209 意味着只有 10 个端口，1 个 CLB 配额 50 监听器足够
			// 但 endPort 限制了可用端口范围，第一个 CLB 端口耗尽后会触发扩容
			endPort := uint16(31209)
			spec.EndPort = &endPort
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "scaleup"
			createDep(ctx, depName, mappingAnnotations(80, "TCPUDP", poolName), 5, 80)
			defer deleteDep(ctx, depName)

			waitAllMappingsReady(ctx, depName, 5)

			// 验证端口池中有 CLB 被自动创建
			Eventually(func() int {
				updated := &networkingv1alpha1.CLBPortPool{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: poolName}, updated)
				return len(updated.Status.LoadbalancerStatuses)
			}, maxWait, pollInterval).Should(BeNumerically(">", 0), "端口池应该有自动创建的 CLB")
		})
	})

	// 13. segmentLength（端口段）映射
	Describe("端口段映射", func() {
		It("应支持 segmentLength 端口段映射", func() {
			poolName := poolPrefix + "segment"
			spec := autoCreateLBSpec(31300)
			segLen := uint16(10)
			spec.SegmentLength = &segLen
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "segment"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
			Expect(mappings[0].LoadbalancerEndPort).NotTo(BeNil(),
				"端口段映射应该有 LoadbalancerEndPort")
			endPort := *mappings[0].LoadbalancerEndPort
			Expect(endPort - mappings[0].LoadbalancerPort).To(Equal(uint16(9)),
				"端口段长度为 10，EndPort-Port 应为 9")
		})
	})

	// 14. LB 黑名单
	Describe("LB 黑名单", func() {
		It("应支持 LB 黑名单功能", func() {
			poolName := poolPrefix + "blacklist"
			spec := autoCreateLBSpec(31400)
			// 先不设黑名单，创建一个 Pod 让控制器自动创建 CLB
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName1 := appPrefix + "blacklist-1"
			createDep(ctx, depName1, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName1)
			mappings := waitMappingReady(ctx, depName1)
			Expect(mappings).NotTo(BeEmpty())
			lbID := mappings[0].LoadbalancerId

			// 将该 CLB 加入黑名单
			pool := &networkingv1alpha1.CLBPortPool{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: poolName}, pool)).To(Succeed())
			pool.Spec.LbBlacklist = []string{lbID}
			Expect(k8sClient.Update(ctx, pool)).To(Succeed())

			// 等待端口池 reconcile 完成，确保黑名单生效
			Eventually(func() bool {
				updated := &networkingv1alpha1.CLBPortPool{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: poolName}, updated); err != nil {
					return false
				}
				return updated.Status.State == networkingv1alpha1.CLBPortPoolStateActive
			}, maxWait, pollInterval).Should(BeTrue(), "黑名单更新后端口池应恢复 Active")

			// 创建新 Pod，应该分配到不同的 CLB（自动创建新的）
			depName2 := appPrefix + "blacklist-2"
			createDep(ctx, depName2, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName2)
			mappings2 := waitMappingReady(ctx, depName2)
			Expect(mappings2).NotTo(BeEmpty())
			Expect(mappings2[0].LoadbalancerId).NotTo(Equal(lbID),
				"黑名单中的 CLB 不应被再次分配")
		})
	})

	// 15. InOrder 分配策略
	Describe("InOrder 分配策略", func() {
		It("应支持 InOrder 分配策略", func() {
			poolName := poolPrefix + "policy-inorder"
			policy := "InOrder"
			spec := autoCreateLBSpec(31500)
			spec.LbPolicy = &policy
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "policy-inorder"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
		})
	})

	// 16. Random 分配策略
	Describe("Random 分配策略", func() {
		It("应支持 Random 分配策略", func() {
			poolName := poolPrefix + "policy-random"
			policy := "Random"
			spec := autoCreateLBSpec(31600)
			spec.LbPolicy = &policy
			createPool(ctx, poolName, spec)
			defer deletePool(ctx, poolName)
			waitPoolActive(ctx, poolName)

			depName := appPrefix + "policy-random"
			createDep(ctx, depName, mappingAnnotations(80, "TCP", poolName), 1, 80)
			defer deleteDep(ctx, depName)

			mappings := waitMappingReady(ctx, depName)
			Expect(mappings).NotTo(BeEmpty())
		})
	})
})

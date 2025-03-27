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

package controller

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/clbbinding"
	"github.com/imroc/tke-extend-network-controller/internal/constant"
	"github.com/imroc/tke-extend-network-controller/pkg/eventsource"
	"github.com/imroc/tke-extend-network-controller/pkg/kube"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	CLBBindingReconciler[*clbbinding.CLBPodBinding]
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// return ReconcilePodWithFinalizer(ctx, req, r.Client, &corev1.Pod{}, r.sync, r.cleanup)
	return Reconcile(ctx, req, r.Client, &corev1.Pod{}, r.sync)
}

func (r *PodReconciler) sync(ctx context.Context, pod *corev1.Pod) (result ctrl.Result, err error) {
	// 获取 obj 的注解
	if pod.Annotations[constant.EnableCLBPortMappingsKey] != "" {
		result, err = r.syncCLBBinding(ctx, pod, clbbinding.NewCLBPodBinding())
		if err != nil {
			return result, errors.WithStack(err)
		}
		if result.Requeue || result.RequeueAfter > 0 { // 重新入队
			return result, nil
		}
	}
	if pod.Annotations[constant.EnableCLBHostPortMapping] == "true" {
		if pod.Spec.NodeName == "" {
			log.FromContext(ctx).V(5).Info("skip host port mapping when pod not schedued to node")
			return
		}
		result, err = r.syncCLBHostPortMapping(ctx, pod)
		if err != nil {
			return result, errors.WithStack(err)
		}
	}
	return
}

type HostPortMapping struct {
	// 应用端口
	ContainerPort uint16 `json:"containerPort"`
	// 主机端口
	HostPort uint16 `json:"hostPort"`
	// 协议类型
	Protocol string `json:"protocol"`
	// 使用的端口池
	Pool string `json:"pool"`
	// 地域信息
	Region string `json:"region"`
	// 负载均衡器ID
	LoadbalancerId string `json:"loadbalancerId"`
	// 负载均衡器端口
	LoadbalancerPort uint16 `json:"loadbalancerPort"`
	// 监听器ID
	ListenerId string `json:"listenerId"`
}

func matchHostPort(hostPort, port, lbPort, lbEndPort uint16) (uint16, bool) {
	if lbEndPort > 0 {
		endPort := port + (lbEndPort - lbPort)
		if hostPort >= port && hostPort <= endPort { // 命中端口段
			return lbPort + (hostPort - port), true
		} else {
			return 0, false
		}
	}
	if hostPort == port { // 命中单端口监听器
		return lbPort, true
	}
	return 0, false
}

func (r *PodReconciler) syncCLBHostPortMapping(ctx context.Context, pod *corev1.Pod) (result ctrl.Result, err error) {
	cbd := &networkingv1alpha1.CLBNodeBinding{}
	if err = r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, cbd); err != nil {
		return result, errors.WithStack(err)
	}
	mappings := []HostPortMapping{}
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.HostPort == 0 {
				continue
			}
			for _, bd := range cbd.Status.PortBindings {
				if bd.Protocol == string(port.Protocol) {
					if lbPort, ok := matchHostPort(uint16(port.HostPort), bd.Port, bd.LoadbalancerPort, util.GetValue(bd.LoadbalancerEndPort)); ok {
						mappings = append(mappings, HostPortMapping{
							ContainerPort:    uint16(port.ContainerPort),
							HostPort:         uint16(port.HostPort),
							Protocol:         bd.Protocol,
							Pool:             bd.Pool,
							Region:           bd.Region,
							LoadbalancerId:   bd.LoadbalancerId,
							LoadbalancerPort: lbPort,
							ListenerId:       bd.ListenerId,
						})
					}
				}
			}
			if len(mappings) == 0 {
				log.FromContext(ctx).V(5).Info("no hostport avaliable, make sure you enabled the clb port binding to the node")
				return
			}
			val, err := json.Marshal(mappings)
			if err != nil {
				return result, errors.WithStack(err)
			}
			if pod.Annotations[constant.CLBHostPortMappingResultKey] != string(val) {
				patchMap := map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]string{
							constant.CLBHostPortMappingResultKey:  string(val),
							constant.CLBHostPortMappingStatuslKey: "Ready",
						},
					},
				}
				if err := kube.PatchMap(ctx, r.Client, pod, patchMap); err != nil {
					return result, errors.WithStack(err)
				}
			}
		}
	}
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager, workers int) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		WatchesRawSource(source.Channel(eventsource.Pod, &handler.EnqueueRequestForObject{})).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workers,
		}).
		Named("pod").
		Complete(r)
}

func (r *PodReconciler) findObjectsForPod(_ context.Context, obj client.Object) []reconcile.Request {
	if anno := obj.GetAnnotations(); anno != nil && (anno[constant.EnableCLBPortMappingsKey] != "" || anno[constant.EnableCLBHostPortMapping] != "") {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
				},
			},
		}
	}
	return nil
}

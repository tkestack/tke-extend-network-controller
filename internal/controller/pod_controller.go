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
	"bufio"
	"context"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	networkingv1alpha1 "github.com/imroc/tke-extend-network-controller/api/v1alpha1"
	"github.com/imroc/tke-extend-network-controller/internal/constant"
	"github.com/imroc/tke-extend-network-controller/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
	return ReconcileWithResult(ctx, req, r.Client, &corev1.Pod{}, r.sync)
}

func (r *PodReconciler) sync(ctx context.Context, pod *corev1.Pod) (result ctrl.Result, err error) {
	if !pod.DeletionTimestamp.IsZero() { // 忽略正在删除的 Pod
		return
	}
	// 获取 Pod 的注解
	enablePortMappings := pod.Annotations[constant.EnableCLBPortMappingsKey]
	if enablePortMappings == "" {
		log.FromContext(ctx).Info("skip pod without enable-clb-port-mapping annotation")
		return
	}

	portMappings := pod.Annotations[constant.CLBPortMappingsKey]
	if portMappings == "" {
		log.FromContext(ctx).Info("skip pod without clb-port-mapping annotation")
		return
	}

	switch enablePortMappings {
	case "true", "false": // 确保 CLBPodBinding 存在且符合预期
		// 获取 Pod 对应的 CLBPodBinding
		pb := &networkingv1alpha1.CLBPodBinding{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(pod), pb); err != nil {
			if apierrors.IsNotFound(err) { // 不存在，自动创建
				// 没有 CLBPodBinding，自动创建
				pb.Name = pod.Name
				pb.Namespace = pod.Namespace
				// 生成期望的 CLBPodBindingSpec
				spec, err := generateCLBPodBindingSpec(portMappings, enablePortMappings)
				if err != nil {
					return result, errors.Wrap(err, "failed to generate CLBPodBinding spec")
				}
				pb.Spec = *spec
				if pod.Annotations[constant.Ratain] != "true" { // 添加 Owner
					if err := controllerutil.SetOwnerReference(pod, pb, r.Scheme); err != nil {
						return result, errors.WithStack(err)
					}
				}
				log.FromContext(ctx).Info("create clbpodbinding", "pb", *pb)
				if err := r.Create(ctx, pb); err != nil {
					return result, errors.WithStack(err)
				}
			} else { // 其它错误，直接返回错误
				return result, errors.WithStack(err)
			}
		} else { // 存在
			// 正在删除，重新入队（滚动更新场景，旧的解绑完，确保 CLBPodBinding 要重建出来）
			if !pb.DeletionTimestamp.IsZero() {
				result.RequeueAfter = time.Second
				return result, nil
			}
			// CLBPodBinding 存在且没有被删除，对账 spec 是否符合预期
			spec, err := generateCLBPodBindingSpec(portMappings, enablePortMappings)
			if err != nil {
				return result, errors.Wrap(err, "failed to generate CLBPodBinding spec")
			}
			if !reflect.DeepEqual(pb.Spec, spec) { // spec 不一致，更新
				pb.Spec = *spec
				log.FromContext(ctx).Info("update clbpodbinding")
				if err := r.Update(ctx, pb); err != nil {
					return result, errors.WithStack(err)
				}
			}
		}
	default:
		log.FromContext(ctx).Info("skip invalid enable-clb-port-mapping value", "value", enablePortMappings)
	}
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPod),
		).
		Named("pod").
		Complete(r)
}

// 过滤带有 networking.cloud.tencent.com/enable-clb-port-mapping 注解的 Pod
func (r *PodReconciler) findObjectsForPod(ctx context.Context, pod client.Object) []reconcile.Request {
	if pod.GetAnnotations()[constant.EnableCLBPortMappingsKey] == "" {
		return []reconcile.Request{}
	}
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      pod.GetName(),
				Namespace: pod.GetNamespace(),
			},
		},
	}
}

func generateCLBPodBindingSpec(anno, enablePortMappings string) (*networkingv1alpha1.CLBPodBindingSpec, error) {
	spec := &networkingv1alpha1.CLBPodBindingSpec{}
	rd := bufio.NewReader(strings.NewReader(anno))
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		fields := strings.Fields(string(line))
		if len(fields) < 3 {
			return nil, fmt.Errorf("invalid port mapping: %s", string(line))
		}
		portStr := fields[0]
		portUint64, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("bad port number in port mapping: %s", string(line))
		}
		port := uint16(portUint64)
		protocol := fields[1]
		pools := strings.Split(fields[2], ",")
		var useSamePortAcrossPools *bool
		if len(fields) >= 4 && fields[3] == "useSamePortAcrossPools" {
			b := true
			useSamePortAcrossPools = &b
		}
		spec.Ports = append(spec.Ports, networkingv1alpha1.PortEntry{
			Port:                   port,
			Protocol:               protocol,
			Pools:                  pools,
			UseSamePortAcrossPools: useSamePortAcrossPools,
		})
	}
	if enablePortMappings == "false" {
		spec.Disabled = util.GetPtr(true)
	}
	return spec, nil
}

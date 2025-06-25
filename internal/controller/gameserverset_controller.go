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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/constant"
	"github.com/tkestack/tke-extend-network-controller/pkg/util"
	gamekruiseiov1alpha1 "github.com/openkruise/kruise-game/apis/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GameServerSetReconciler reconciles a GameServerSet object
type GameServerSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=game.kruise.io,resources=gameserversets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=game.kruise.io,resources=gameserversets/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GameServerSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ReconcileWithFinalizer(ctx, req, r.Client, &gamekruiseiov1alpha1.GameServerSet{}, r.sync, r.cleanup)
}

func (r *GameServerSetReconciler) cleanup(ctx context.Context, gss *gamekruiseiov1alpha1.GameServerSet) (ctrl.Result, error) {
	ppName := getPortPoolName(gss)
	pp := &networkingv1alpha1.CLBPortPool{}
	if err := r.Get(ctx, client.ObjectKey{Name: ppName}, pp); err != nil {
		if apierrors.IsNotFound(err) { // 已经删除，忽略
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.WithStack(err)
	}
	if !pp.DeletionTimestamp.IsZero() { // 正在删除，忽略
		return ctrl.Result{}, nil
	}
	// 还未删除，发起删除请求
	if err := r.Delete(ctx, pp); err != nil {
		return ctrl.Result{}, errors.WithStack(err)
	}
	return ctrl.Result{}, nil
}

func getPortPoolName(gss *gamekruiseiov1alpha1.GameServerSet) string {
	return fmt.Sprintf("%s-%s", gss.Namespace, gss.Name)
}

func (r *GameServerSetReconciler) sync(ctx context.Context, gss *gamekruiseiov1alpha1.GameServerSet) (ctrl.Result, error) {
	network := gss.Spec.Network
	if network == nil {
		return ctrl.Result{}, nil
	}
	if network.NetworkType != constant.OKGNetworkType {
		return ctrl.Result{}, nil
	}
	var clbIds []string
	var startPort uint16 = 30000
	var listenerQuota *uint16
	for _, param := range network.NetworkConf {
		switch param.Name {
		case "ClbIds":
			clbIds = strings.Split(param.Value, ",")
		case "MinPort":
			if minPort, err := strconv.Atoi(param.Value); err != nil {
				return ctrl.Result{}, errors.WithStack(err)
			} else {
				startPort = uint16(minPort)
			}
		case "ListenerQuota":
			if quota, err := strconv.Atoi(param.Value); err != nil {
				return ctrl.Result{}, errors.WithStack(err)
			} else {
				listenerQuota = util.GetPtr(uint16(quota))
			}
		}
	}

	// 构造期望的 CLBPortPool Spec
	ppSpec := networkingv1alpha1.CLBPortPoolSpec{
		StartPort:               startPort,
		ListenerQuota:           listenerQuota,
		ExsistedLoadBalancerIDs: clbIds,
	}

	// 确保 CLBPortPool 存在并符合预期
	ppName := getPortPoolName(gss)
	pp := &networkingv1alpha1.CLBPortPool{}
	if err := r.Get(ctx, client.ObjectKey{Name: ppName}, pp); err != nil {
		if apierrors.IsNotFound(err) { // 不存在，创建一个
			pp.Name = ppName
			pp.Spec = ppSpec
			if err := r.Create(ctx, pp); err != nil {
				return ctrl.Result{}, errors.WithStack(err)
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.WithStack(err)
	}
	// 存在，检查是否符合预期
	if !reflect.DeepEqual(pp.Spec, ppSpec) { // 不符合预期，更新
		pp.Spec = ppSpec
		if err := r.Update(ctx, pp); err != nil {
			return ctrl.Result{}, errors.WithStack(err)
		}
	}
	// 全部符合预期，忽略
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameServerSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gamekruiseiov1alpha1.GameServerSet{}).
		Named("gameserverset").
		Complete(r)
}

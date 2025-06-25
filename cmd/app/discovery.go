package app

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	gamekruiseiov1alpha1 "github.com/openkruise/kruise-game/apis/v1alpha1"
	"github.com/pkg/errors"
	"github.com/tkestack/tke-extend-network-controller/pkg/clusterinfo"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// 手动创建 RESTMapper
func createRESTMapper(config *rest.Config) (meta.RESTMapper, error) {
	// 1. 创建 Discovery 客户端
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	// 2. 创建带内存缓存的 Discovery 客户端（提高性能）
	cachedClient := memory.NewMemCacheClient(dc)

	// 3. 创建延迟加载的 RESTMapper
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)

	return mapper, nil
}

func InitOptionalScheme(config *rest.Config) error {
	restMapper, err := createRESTMapper(config)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := initAgonesScheme(restMapper); err != nil {
		return errors.WithStack(err)
	}
	if err := initOKGScheme(restMapper); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func initAgonesScheme(restMapper meta.RESTMapper) error {
	agonesGVK := schema.GroupVersionKind{
		Group:   "agones.dev",
		Version: "v1",
		Kind:    "GameServer",
	}
	// 检查 CRD 是否存在
	if installed, err := isCRDInstalled(restMapper, agonesGVK); err != nil {
		return errors.WithStack(err)
	} else if installed {
		utilruntime.Must(agonesv1.AddToScheme(scheme))
		clusterinfo.AgonesSupported = true
		setupLog.Info("agones CRD discovered, will enable agones support")
	}
	return nil
}

func initOKGScheme(restMapper meta.RESTMapper) error {
	okgGVK := schema.GroupVersionKind{
		Group:   "game.kruise.io",
		Version: "v1alpha1",
		Kind:    "GameServerSet",
	}
	// 检查 CRD 是否存在
	if installed, err := isCRDInstalled(restMapper, okgGVK); err != nil {
		return errors.WithStack(err)
	} else if installed {
		utilruntime.Must(gamekruiseiov1alpha1.AddToScheme(scheme))
		clusterinfo.OKGSupported = true
		setupLog.Info("OpenKruiseGame CRD discovered, will enable OpenKruiseGame support")
	}
	return nil
}

// 检查 CRD 是否存在的函数
func isCRDInstalled(restMapper meta.RESTMapper, gvk schema.GroupVersionKind) (bool, error) {
	_, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		if meta.IsNoMatchError(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

## kubebuilder 版本

当前使用的 kubebuilder 版本为 4.5.2。

## kubebuilder 使用记录

### v1alpha1

```bash
kubebuilder create api --group networking --version v1alpha1 --kind DedicatedCLBListener --namespaced=true --resource --controller
kubebuilder create webhook --group networking --version v1alpha1 --kind DedicatedCLBListener --defaulting --programmatic-validation

kubebuilder create api --group networking --version v1alpha1 --kind DedicatedCLBService --namespaced=true --resource --controller

kubebuilder create api --group networking --version v1alpha1 --kind CLBPodBinding --namespaced=true --resource --controller
kubebuilder create api --group networking --version v1alpha1 --kind CLBNodeBinding --namespaced=false --resource --controller
kubebuilder create api --group networking --version v1alpha1 --kind CLBPortPool --namespaced=false --resource --controller

kubebuilder create webhook --group networking --version v1alpha1 --kind CLBPortPool --defaulting --programmatic-validation
kubebuilder create webhook --group networking --version v1alpha1 --kind CLBPodBinding --programmatic-validation

kubebuilder create api --group core --kind Pod --version v1 --controller=true --resource=false
kubebuilder create webhook --group core --version v1 --kind Pod --defaulting --programmatic-validation

kubebuilder create api --group core --kind Node --version v1 --controller=true --resource=false
```

//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AllocatableCLBInfo) DeepCopyInto(out *AllocatableCLBInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AllocatableCLBInfo.
func (in *AllocatableCLBInfo) DeepCopy() *AllocatableCLBInfo {
	if in == nil {
		return nil
	}
	out := new(AllocatableCLBInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AllocatedCLBInfo) DeepCopyInto(out *AllocatedCLBInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AllocatedCLBInfo.
func (in *AllocatedCLBInfo) DeepCopy() *AllocatedCLBInfo {
	if in == nil {
		return nil
	}
	out := new(AllocatedCLBInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AutoCreateConfig) DeepCopyInto(out *AutoCreateConfig) {
	*out = *in
	if in.MaxLoadBalancers != nil {
		in, out := &in.MaxLoadBalancers, &out.MaxLoadBalancers
		*out = new(uint16)
		**out = **in
	}
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = new(CreateLBParameters)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AutoCreateConfig.
func (in *AutoCreateConfig) DeepCopy() *AutoCreateConfig {
	if in == nil {
		return nil
	}
	out := new(AutoCreateConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBNodeBinding) DeepCopyInto(out *CLBNodeBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBNodeBinding.
func (in *CLBNodeBinding) DeepCopy() *CLBNodeBinding {
	if in == nil {
		return nil
	}
	out := new(CLBNodeBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CLBNodeBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBNodeBindingList) DeepCopyInto(out *CLBNodeBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CLBNodeBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBNodeBindingList.
func (in *CLBNodeBindingList) DeepCopy() *CLBNodeBindingList {
	if in == nil {
		return nil
	}
	out := new(CLBNodeBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CLBNodeBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBNodeBindingSpec) DeepCopyInto(out *CLBNodeBindingSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBNodeBindingSpec.
func (in *CLBNodeBindingSpec) DeepCopy() *CLBNodeBindingSpec {
	if in == nil {
		return nil
	}
	out := new(CLBNodeBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBNodeBindingStatus) DeepCopyInto(out *CLBNodeBindingStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBNodeBindingStatus.
func (in *CLBNodeBindingStatus) DeepCopy() *CLBNodeBindingStatus {
	if in == nil {
		return nil
	}
	out := new(CLBNodeBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPodBinding) DeepCopyInto(out *CLBPodBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPodBinding.
func (in *CLBPodBinding) DeepCopy() *CLBPodBinding {
	if in == nil {
		return nil
	}
	out := new(CLBPodBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CLBPodBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPodBindingList) DeepCopyInto(out *CLBPodBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CLBPodBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPodBindingList.
func (in *CLBPodBindingList) DeepCopy() *CLBPodBindingList {
	if in == nil {
		return nil
	}
	out := new(CLBPodBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CLBPodBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPodBindingSpec) DeepCopyInto(out *CLBPodBindingSpec) {
	*out = *in
	if in.Disabled != nil {
		in, out := &in.Disabled, &out.Disabled
		*out = new(bool)
		**out = **in
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]PortEntry, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPodBindingSpec.
func (in *CLBPodBindingSpec) DeepCopy() *CLBPodBindingSpec {
	if in == nil {
		return nil
	}
	out := new(CLBPodBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPodBindingStatus) DeepCopyInto(out *CLBPodBindingStatus) {
	*out = *in
	if in.PortBindings != nil {
		in, out := &in.PortBindings, &out.PortBindings
		*out = make([]PortBindingStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPodBindingStatus.
func (in *CLBPodBindingStatus) DeepCopy() *CLBPodBindingStatus {
	if in == nil {
		return nil
	}
	out := new(CLBPodBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPortPool) DeepCopyInto(out *CLBPortPool) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPortPool.
func (in *CLBPortPool) DeepCopy() *CLBPortPool {
	if in == nil {
		return nil
	}
	out := new(CLBPortPool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CLBPortPool) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPortPoolList) DeepCopyInto(out *CLBPortPoolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CLBPortPool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPortPoolList.
func (in *CLBPortPoolList) DeepCopy() *CLBPortPoolList {
	if in == nil {
		return nil
	}
	out := new(CLBPortPoolList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CLBPortPoolList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPortPoolSpec) DeepCopyInto(out *CLBPortPoolSpec) {
	*out = *in
	if in.EndPort != nil {
		in, out := &in.EndPort, &out.EndPort
		*out = new(uint16)
		**out = **in
	}
	if in.SegmentLength != nil {
		in, out := &in.SegmentLength, &out.SegmentLength
		*out = new(uint16)
		**out = **in
	}
	if in.Region != nil {
		in, out := &in.Region, &out.Region
		*out = new(string)
		**out = **in
	}
	if in.ExsistedLoadBalancerIDs != nil {
		in, out := &in.ExsistedLoadBalancerIDs, &out.ExsistedLoadBalancerIDs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AutoCreate != nil {
		in, out := &in.AutoCreate, &out.AutoCreate
		*out = new(AutoCreateConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPortPoolSpec.
func (in *CLBPortPoolSpec) DeepCopy() *CLBPortPoolSpec {
	if in == nil {
		return nil
	}
	out := new(CLBPortPoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLBPortPoolStatus) DeepCopyInto(out *CLBPortPoolStatus) {
	*out = *in
	if in.State != nil {
		in, out := &in.State, &out.State
		*out = new(string)
		**out = **in
	}
	if in.Message != nil {
		in, out := &in.Message, &out.Message
		*out = new(string)
		**out = **in
	}
	if in.LoadbalancerStatuses != nil {
		in, out := &in.LoadbalancerStatuses, &out.LoadbalancerStatuses
		*out = make([]LoadBalancerStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLBPortPoolStatus.
func (in *CLBPortPoolStatus) DeepCopy() *CLBPortPoolStatus {
	if in == nil {
		return nil
	}
	out := new(CLBPortPoolStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CreateLBParameters) DeepCopyInto(out *CreateLBParameters) {
	*out = *in
	if in.VipIsp != nil {
		in, out := &in.VipIsp, &out.VipIsp
		*out = new(string)
		**out = **in
	}
	if in.BandwidthPackageId != nil {
		in, out := &in.BandwidthPackageId, &out.BandwidthPackageId
		*out = new(string)
		**out = **in
	}
	if in.AddressIPVersion != nil {
		in, out := &in.AddressIPVersion, &out.AddressIPVersion
		*out = new(string)
		**out = **in
	}
	if in.LoadBalancerPassToTarget != nil {
		in, out := &in.LoadBalancerPassToTarget, &out.LoadBalancerPassToTarget
		*out = new(bool)
		**out = **in
	}
	if in.DynamicVip != nil {
		in, out := &in.DynamicVip, &out.DynamicVip
		*out = new(bool)
		**out = **in
	}
	if in.VpcId != nil {
		in, out := &in.VpcId, &out.VpcId
		*out = new(string)
		**out = **in
	}
	if in.Vip != nil {
		in, out := &in.Vip, &out.Vip
		*out = new(string)
		**out = **in
	}
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]TagInfo, len(*in))
		copy(*out, *in)
	}
	if in.ProjectId != nil {
		in, out := &in.ProjectId, &out.ProjectId
		*out = new(int64)
		**out = **in
	}
	if in.LoadBalancerName != nil {
		in, out := &in.LoadBalancerName, &out.LoadBalancerName
		*out = new(string)
		**out = **in
	}
	if in.LoadBalancerType != nil {
		in, out := &in.LoadBalancerType, &out.LoadBalancerType
		*out = new(string)
		**out = **in
	}
	if in.MasterZoneId != nil {
		in, out := &in.MasterZoneId, &out.MasterZoneId
		*out = new(string)
		**out = **in
	}
	if in.ZoneId != nil {
		in, out := &in.ZoneId, &out.ZoneId
		*out = new(string)
		**out = **in
	}
	if in.SubnetId != nil {
		in, out := &in.SubnetId, &out.SubnetId
		*out = new(string)
		**out = **in
	}
	if in.SlaType != nil {
		in, out := &in.SlaType, &out.SlaType
		*out = new(string)
		**out = **in
	}
	if in.LBChargeType != nil {
		in, out := &in.LBChargeType, &out.LBChargeType
		*out = new(string)
		**out = **in
	}
	if in.InternetAccessible != nil {
		in, out := &in.InternetAccessible, &out.InternetAccessible
		*out = new(InternetAccessible)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CreateLBParameters.
func (in *CreateLBParameters) DeepCopy() *CreateLBParameters {
	if in == nil {
		return nil
	}
	out := new(CreateLBParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBListener) DeepCopyInto(out *DedicatedCLBListener) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBListener.
func (in *DedicatedCLBListener) DeepCopy() *DedicatedCLBListener {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBListener)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DedicatedCLBListener) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBListenerList) DeepCopyInto(out *DedicatedCLBListenerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DedicatedCLBListener, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBListenerList.
func (in *DedicatedCLBListenerList) DeepCopy() *DedicatedCLBListenerList {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBListenerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DedicatedCLBListenerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBListenerSpec) DeepCopyInto(out *DedicatedCLBListenerSpec) {
	*out = *in
	if in.TargetPod != nil {
		in, out := &in.TargetPod, &out.TargetPod
		*out = new(TargetPod)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBListenerSpec.
func (in *DedicatedCLBListenerSpec) DeepCopy() *DedicatedCLBListenerSpec {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBListenerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBListenerStatus) DeepCopyInto(out *DedicatedCLBListenerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBListenerStatus.
func (in *DedicatedCLBListenerStatus) DeepCopy() *DedicatedCLBListenerStatus {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBListenerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBService) DeepCopyInto(out *DedicatedCLBService) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBService.
func (in *DedicatedCLBService) DeepCopy() *DedicatedCLBService {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DedicatedCLBService) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBServiceList) DeepCopyInto(out *DedicatedCLBServiceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DedicatedCLBService, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBServiceList.
func (in *DedicatedCLBServiceList) DeepCopy() *DedicatedCLBServiceList {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBServiceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DedicatedCLBServiceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBServicePort) DeepCopyInto(out *DedicatedCLBServicePort) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBServicePort.
func (in *DedicatedCLBServicePort) DeepCopy() *DedicatedCLBServicePort {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBServicePort)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBServiceSpec) DeepCopyInto(out *DedicatedCLBServiceSpec) {
	*out = *in
	if in.VpcId != nil {
		in, out := &in.VpcId, &out.VpcId
		*out = new(string)
		**out = **in
	}
	if in.MaxPod != nil {
		in, out := &in.MaxPod, &out.MaxPod
		*out = new(int64)
		**out = **in
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]DedicatedCLBServicePort, len(*in))
		copy(*out, *in)
	}
	if in.ExistedLbIds != nil {
		in, out := &in.ExistedLbIds, &out.ExistedLbIds
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.LbAutoCreate = in.LbAutoCreate
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBServiceSpec.
func (in *DedicatedCLBServiceSpec) DeepCopy() *DedicatedCLBServiceSpec {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBServiceStatus) DeepCopyInto(out *DedicatedCLBServiceStatus) {
	*out = *in
	if in.AllocatableLb != nil {
		in, out := &in.AllocatableLb, &out.AllocatableLb
		*out = make([]AllocatableCLBInfo, len(*in))
		copy(*out, *in)
	}
	if in.AllocatedLb != nil {
		in, out := &in.AllocatedLb, &out.AllocatedLb
		*out = make([]AllocatedCLBInfo, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DedicatedCLBServiceStatus.
func (in *DedicatedCLBServiceStatus) DeepCopy() *DedicatedCLBServiceStatus {
	if in == nil {
		return nil
	}
	out := new(DedicatedCLBServiceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternetAccessible) DeepCopyInto(out *InternetAccessible) {
	*out = *in
	if in.InternetChargeType != nil {
		in, out := &in.InternetChargeType, &out.InternetChargeType
		*out = new(string)
		**out = **in
	}
	if in.InternetMaxBandwidthOut != nil {
		in, out := &in.InternetMaxBandwidthOut, &out.InternetMaxBandwidthOut
		*out = new(int64)
		**out = **in
	}
	if in.BandwidthpkgSubType != nil {
		in, out := &in.BandwidthpkgSubType, &out.BandwidthpkgSubType
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternetAccessible.
func (in *InternetAccessible) DeepCopy() *InternetAccessible {
	if in == nil {
		return nil
	}
	out := new(InternetAccessible)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LbAutoCreate) DeepCopyInto(out *LbAutoCreate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LbAutoCreate.
func (in *LbAutoCreate) DeepCopy() *LbAutoCreate {
	if in == nil {
		return nil
	}
	out := new(LbAutoCreate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadBalancerStatus) DeepCopyInto(out *LoadBalancerStatus) {
	*out = *in
	if in.AutoCreated != nil {
		in, out := &in.AutoCreated, &out.AutoCreated
		*out = new(bool)
		**out = **in
	}
	if in.Ips != nil {
		in, out := &in.Ips, &out.Ips
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Hostname != nil {
		in, out := &in.Hostname, &out.Hostname
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadBalancerStatus.
func (in *LoadBalancerStatus) DeepCopy() *LoadBalancerStatus {
	if in == nil {
		return nil
	}
	out := new(LoadBalancerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PortBindingStatus) DeepCopyInto(out *PortBindingStatus) {
	*out = *in
	if in.LoadbalancerEndPort != nil {
		in, out := &in.LoadbalancerEndPort, &out.LoadbalancerEndPort
		*out = new(uint16)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PortBindingStatus.
func (in *PortBindingStatus) DeepCopy() *PortBindingStatus {
	if in == nil {
		return nil
	}
	out := new(PortBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PortEntry) DeepCopyInto(out *PortEntry) {
	*out = *in
	if in.Pools != nil {
		in, out := &in.Pools, &out.Pools
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.UseSamePortAcrossPools != nil {
		in, out := &in.UseSamePortAcrossPools, &out.UseSamePortAcrossPools
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PortEntry.
func (in *PortEntry) DeepCopy() *PortEntry {
	if in == nil {
		return nil
	}
	out := new(PortEntry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TagInfo) DeepCopyInto(out *TagInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TagInfo.
func (in *TagInfo) DeepCopy() *TagInfo {
	if in == nil {
		return nil
	}
	out := new(TagInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetPod) DeepCopyInto(out *TargetPod) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetPod.
func (in *TargetPod) DeepCopy() *TargetPod {
	if in == nil {
		return nil
	}
	out := new(TargetPod)
	in.DeepCopyInto(out)
	return out
}

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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CLB) DeepCopyInto(out *CLB) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CLB.
func (in *CLB) DeepCopy() *CLB {
	if in == nil {
		return nil
	}
	out := new(CLB)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DedicatedCLBListener) DeepCopyInto(out *DedicatedCLBListener) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
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
	if in.CLBs != nil {
		in, out := &in.CLBs, &out.CLBs
		*out = make([]CLB, len(*in))
		copy(*out, *in)
	}
	if in.EndPort != nil {
		in, out := &in.EndPort, &out.EndPort
		*out = new(int64)
		**out = **in
	}
	if in.ExtensiveParameters != nil {
		in, out := &in.ExtensiveParameters, &out.ExtensiveParameters
		*out = new(string)
		**out = **in
	}
	if in.TargetPod != nil {
		in, out := &in.TargetPod, &out.TargetPod
		*out = new(TargetPod)
		**out = **in
	}
	if in.TargetNode != nil {
		in, out := &in.TargetNode, &out.TargetNode
		*out = new(TargetNode)
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
	if in.ListenerStatuses != nil {
		in, out := &in.ListenerStatuses, &out.ListenerStatuses
		*out = make([]ListenerStatus, len(*in))
		copy(*out, *in)
	}
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
func (in *ListenerStatus) DeepCopyInto(out *ListenerStatus) {
	*out = *in
	out.CLB = in.CLB
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListenerStatus.
func (in *ListenerStatus) DeepCopy() *ListenerStatus {
	if in == nil {
		return nil
	}
	out := new(ListenerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetNode) DeepCopyInto(out *TargetNode) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetNode.
func (in *TargetNode) DeepCopy() *TargetNode {
	if in == nil {
		return nil
	}
	out := new(TargetNode)
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

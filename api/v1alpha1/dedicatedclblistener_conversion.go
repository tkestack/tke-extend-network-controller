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

package v1alpha1

import (
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/imroc/tke-extend-network-controller/api/v1beta1"
)

func convertBackListenerStatuses(statuses []v1beta1.ListenerStatus) []ListenerStatus {
	ret := make([]ListenerStatus, len(statuses))
	for i, status := range statuses {
		ret[i] = ListenerStatus{
			CLB: CLB{
				ID:     status.CLB.ID,
				Region: status.CLB.Region,
			},
			ListenerId: status.ListenerId,
			State:      status.State,
			Message:    status.Message,
			Address:    status.Address,
		}
	}
	return ret
}

func convertListenerStatuses(statuses []ListenerStatus) []v1beta1.ListenerStatus {
	ret := make([]v1beta1.ListenerStatus, len(statuses))
	for i, status := range statuses {
		ret[i] = v1beta1.ListenerStatus{
			CLB: v1beta1.CLB{
				ID:     status.CLB.ID,
				Region: status.CLB.Region,
			},
			ListenerId: status.ListenerId,
			State:      status.State,
			Message:    status.Message,
			Address:    status.Address,
		}
	}
	return ret
}

func convertBackCLBs(clbs []v1beta1.CLB) []CLB {
	ret := make([]CLB, len(clbs))
	for i, clb := range clbs {
		ret[i] = CLB{
			ID:     clb.ID,
			Region: clb.Region,
		}
	}
	return ret
}

func convertCLBs(clbs []CLB) []v1beta1.CLB {
	ret := make([]v1beta1.CLB, len(clbs))
	for i, clb := range clbs {
		ret[i] = v1beta1.CLB{
			ID:     clb.ID,
			Region: clb.Region,
		}
	}
	return ret
}

// ConvertTo converts this DedicatedCLBListener (v1alpha1) to the Hub version (v1beta1).
func (src *DedicatedCLBListener) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.DedicatedCLBListener)
	log.Printf("ConvertTo: Converting DedicatedCLBListener from Spoke version v1alpha1 to Hub version v1beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Port = src.Spec.LbPort
	dst.Spec.EndPort = src.Spec.LbEndPort
	dst.Spec.Protocol = src.Spec.Protocol
	if len(src.Spec.CLBs) > 0 {
		dst.Spec.CLBs = convertCLBs(src.Spec.CLBs)
	} else {
		dst.Spec.CLBs = []v1beta1.CLB{{
			ID:     src.Spec.LbId,
			Region: src.Spec.LbRegion,
		}}
	}
	if pod := src.Spec.TargetPod; pod != nil {
		dst.Spec.TargetPod = &v1beta1.TargetPod{
			PodName: pod.PodName,
			Port:    pod.TargetPort,
		}
	}
	if node := src.Spec.TargetNode; node != nil {
		dst.Spec.TargetNode = &v1beta1.TargetNode{
			NodeName: node.NodeName,
			Port:     node.TargetPort,
		}
	}
	dst.Spec.ExtensiveParameters = src.Spec.ExtensiveParameters
	if len(src.Status.ListenerStatuses) > 0 {
		dst.Status.ListenerStatuses = convertListenerStatuses(src.Status.ListenerStatuses)
	} else {
		dst.Status.ListenerStatuses = []v1beta1.ListenerStatus{
			{
				CLB: v1beta1.CLB{
					ID:     src.Spec.LbId,
					Region: src.Spec.LbRegion,
				},
				ListenerId: src.Status.ListenerId,
				State:      src.Status.State,
				Message:    src.Status.Message,
				Address:    src.Status.Address,
			},
		}
	}
	return nil
}

// ConvertFrom converts the Hub version (v1beta1) to this DedicatedCLBListener (v1alpha1).
func (dst *DedicatedCLBListener) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.DedicatedCLBListener)
	log.Printf("ConvertFrom: Converting DedicatedCLBListener from Hub version v1beta1 to Spoke version v1alpha1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.CLBs = convertBackCLBs(src.Spec.CLBs)
	dst.Spec.LbPort = src.Spec.Port
	dst.Spec.LbEndPort = src.Spec.EndPort
	dst.Spec.Protocol = src.Spec.Protocol
	dst.Spec.ExtensiveParameters = src.Spec.ExtensiveParameters
	if pod := src.Spec.TargetPod; pod != nil {
		dst.Spec.TargetPod = &TargetPod{
			PodName:    pod.PodName,
			TargetPort: pod.Port,
		}
	}
	if node := src.Spec.TargetNode; node != nil {
		dst.Spec.TargetNode = &TargetNode{
			NodeName:   node.NodeName,
			TargetPort: node.Port,
		}
	}
	dst.Status.ListenerStatuses = convertBackListenerStatuses(src.Status.ListenerStatuses)
	return nil
}

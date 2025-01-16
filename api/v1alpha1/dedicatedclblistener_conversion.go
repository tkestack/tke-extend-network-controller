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

	networkingv1beta1 "github.com/imroc/tke-extend-network-controller/api/v1beta1"
)

// ConvertTo converts this DedicatedCLBListener (v1alpha1) to the Hub version (v1beta1).
func (src *DedicatedCLBListener) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*networkingv1beta1.DedicatedCLBListener)
	log.Printf("ConvertTo: Converting DedicatedCLBListener from Spoke version v1alpha1 to Hub version v1beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v1alpha1 to v1beta1
	return nil
}

// ConvertFrom converts the Hub version (v1beta1) to this DedicatedCLBListener (v1alpha1).
func (dst *DedicatedCLBListener) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*networkingv1beta1.DedicatedCLBListener)
	log.Printf("ConvertFrom: Converting DedicatedCLBListener from Hub version v1beta1 to Spoke version v1alpha1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v1beta1 to v1alpha1
	return nil
}

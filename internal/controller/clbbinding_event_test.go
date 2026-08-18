package controller

import (
	"fmt"
	"testing"

	"k8s.io/client-go/tools/record"

	networkingv1alpha1 "github.com/tkestack/tke-extend-network-controller/api/v1alpha1"
	"github.com/tkestack/tke-extend-network-controller/internal/clbbinding"
)

func TestRecordPortBindingRemoved(t *testing.T) {
	recorder := record.NewFakeRecorder(1)
	reconciler := &CLBBindingReconciler[*clbbinding.CLBPodBinding]{
		Recorder: recorder,
	}
	binding := clbbinding.NewCLBPodBinding()
	binding.Name = "test-binding"

	reconciler.recordPortBindingRemoved(binding, &networkingv1alpha1.PortBindingStatus{
		LoadbalancerId:   "lb-test",
		LoadbalancerPort: 8080,
		Protocol:         "TCP",
	})

	select {
	case event := <-recorder.Events:
		want := fmt.Sprintf("%s %s lb %q not exists, remove port binding (lbPort:%d protocol:%s)",
			"Normal", "PortBindingRemoved", "lb-test", 8080, "TCP")
		if event != want {
			t.Fatalf("expect event %q, got %q", want, event)
		}
	default:
		t.Fatal("expect PortBindingRemoved event to be recorded")
	}
}

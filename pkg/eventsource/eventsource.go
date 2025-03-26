package eventsource

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var (
	Pod  = make(chan event.TypedGenericEvent[client.Object])
	Node = make(chan event.TypedGenericEvent[client.Object])
)

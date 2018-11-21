package hooks

import (
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

// STSRolloutHooks is used between statefulset rollout transitions,
// to prepare the cluster for the upgrade and validate the rollout,
// then to validate or postpone the successive pods updates.
type STSRolloutHooks interface {
	// Name returns the hook's name
	Name() string

	// PodUpdateTransition is called between pods updates.
	// prev is the previously updated pod (or nil when we're starting a new rollout).
	// next is the pod we're about to update (or nil when we just updated the last pod).
	// If PodUpdateTransition returns true, the controller will proceed updating the next pod.
	// If PodUpdateTransition returns false, the controller will postpone the update, and
	// will call PodUpdateTransition again later, until it succeed.
	PodUpdateTransition(logger logr.Logger, sts *appsv1.StatefulSet, prev, next *v1.Pod) bool
}

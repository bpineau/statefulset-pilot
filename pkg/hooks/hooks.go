package hooks

import (
	"k8s.io/api/core/v1"
)

// STSRolloutHooks is called between statefulset pods updates.
// If the hook returns an error, it will be called again later
// until it returns nil; then the next pod is updated.
type STSRolloutHooks interface {
	// Name returns the hook's name
	Name() string

	// PodUpdateTransition is called between pods updates.
	// prev is the previously updated pod (or nil when we're starting a new rollout).
	// next is the pod we're about to update (or nil after we updated the last pod).
	// If PodUpdateTransition returns an error, the controller will postpone the update,
	// and will call PodUpdateTransition again later, until it succeed.
	// If PodUpdateTransition returns nil, the controller will proceed updating the next pod.
	PodUpdateTransition(prev, next *v1.Pod) error
}

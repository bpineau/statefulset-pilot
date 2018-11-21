package hooks

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

// STSRolloutHooks is used between statefulset rollout transitions,
// to prepare the cluster for the upgrade and validate the rollout,
// then to validate or postpone the successive pods updates.
type STSRolloutHooks interface {
	// Name returns the hook's name
	Name() string

	// BeforeSTSRollout is called when a rollout is needed.
	// If this function returns true, the rollout will start right away.
	// Else, BeforeSTSRollout will be called again later.
	BeforeSTSRollout(sts *appsv1.StatefulSet) bool

	// BeforePodUpdate is called after a pod had been updated,
	// and before we update the next pod. If it returns false,
	// we postpone updating the next pod, and will call
	// BeforePodUpdate again later, until we're ready to continue.
	// The pod passed in argument is the pod we just updated.
	// This is NOT called before we update the first statefulset
	// pod (use BeforeSTSRollout() to control the rollout start).
	BeforePodUpdate(pod *v1.Pod) bool

	// Possible future methods (not yet supported):
	AfterPodUpdate(pod *v1.Pod) bool
	AfterSTSRollout(sts *appsv1.StatefulSet) bool
}

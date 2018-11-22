package elasticsearch

import (
	"github.com/bpineau/statefulset-pilot/pkg/hooks"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

type ESHook struct{}

func New() (hooks.STSRolloutHooks, error) {
	return &ESHook{}, nil
}

func (h *ESHook) Name() string {
	return "elasticsearch"
}

func (h *ESHook) PodUpdateTransition(logger logr.Logger, sts *appsv1.StatefulSet, prev, next *v1.Pod) bool {
	// next is nil if we just updated the last pod in the rollout
	if next == nil {
		return afterUpdate(logger, prev)
	}

	// prev is nil if we're about to update the first pod in the rollout
	if prev == nil {
		return beforeUpdate(logger, next)
	}

	return afterUpdate(logger, prev) && beforeUpdate(logger, next)
}

func beforeUpdate(logger logr.Logger, pod *v1.Pod) bool {
	host := pod.Status.PodIP
	name := pod.GetName()

	if err := isGreen(host); err != nil {
		logger.Info("es pod not yet green", "pod", name, "error", err)
		return false
	}

	if err := setAllocation(host, "new_primaries"); err != nil {
		logger.Info("setAllocation=new_primaries failed for es pod", "pod", name)
		return false
	}

	if err := flushSync(host); err != nil {
		logger.Info("flushSync for es pod", "pod", name, "error", err)
		return false
	}

	return true
}

func afterUpdate(logger logr.Logger, pod *v1.Pod) bool {
	host := pod.Status.PodIP
	name := pod.GetName()

	if err := setAllocation(host, "all"); err != nil {
		logger.Info("setAllocation=all failed for es pod", "pod", name)
		return false
	}

	if err := isGreen(host); err != nil {
		logger.Info("es pod not yet green", "pod", name, "error", err)
		return false
	}

	return true
}

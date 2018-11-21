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
	logger.Info("elasticsearch PodUpdateTransition is enjoying this",
		"partition", sts.Spec.UpdateStrategy.RollingUpdate.Partition)
	return true
}

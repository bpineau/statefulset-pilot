package noop

import (
	"github.com/bpineau/statefulset-pilot/pkg/hooks"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

type Hook struct{}

func New() (hooks.STSRolloutHooks, error) {
	return &Hook{}, nil
}

func (h *Hook) Name() string {
	return "noop"
}

func (h *Hook) PodUpdateTransition(logger logr.Logger, sts *appsv1.StatefulSet, prev, next *v1.Pod) bool {
	return true
}

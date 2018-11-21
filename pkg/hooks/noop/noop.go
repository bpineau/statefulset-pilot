package noop

import (
	"github.com/bpineau/statefulset-pilot/pkg/hooks"
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

func (h *Hook) BeforeSTSRollout(sts *appsv1.StatefulSet) bool {
	return true
}

func (h *Hook) AfterSTSRollout(sts *appsv1.StatefulSet) bool {
	return true
}

func (h *Hook) BeforePodUpdate(pod *v1.Pod) bool {
	return true
}

func (h *Hook) AfterPodUpdate(pod *v1.Pod) bool {
	return true
}

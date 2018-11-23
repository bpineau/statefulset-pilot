package noop

import (
	"github.com/bpineau/statefulset-pilot/pkg/hooks"
	"k8s.io/api/core/v1"
)

type Hook struct{}

func New() (hooks.STSRolloutHooks, error) {
	return &Hook{}, nil
}

func (h *Hook) Name() string {
	return "noop"
}

func (h *Hook) PodUpdateTransition(prev, next *v1.Pod) error {
	return nil
}

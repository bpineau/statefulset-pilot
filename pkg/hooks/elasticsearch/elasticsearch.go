package elasticsearch

import (
	"fmt"

	"github.com/bpineau/statefulset-pilot/pkg/hooks"
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

func (h *ESHook) BeforeSTSRollout(sts *appsv1.StatefulSet) bool {
	fmt.Println("ES BeforeSTSRollout called")
	return true
}

func (h *ESHook) BeforePodUpdate(pod *v1.Pod) bool {
	fmt.Println("ES BeforePodUpdate called")
	return true
}

package elasticsearch

import (
	"fmt"

	"github.com/bpineau/statefulset-pilot/pkg/hooks"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
)

type ESHook struct{}

func New() (hooks.STSRolloutHooks, error) {
	return &ESHook{}, nil
}

func (h *ESHook) Name() string {
	return "elasticsearch"
}

func (h *ESHook) PodUpdateTransition(prev, next *v1.Pod) error {
	// prev or next may be nil, when we're on the rollout's first or last pod
	if prev != nil {
		if err := afterUpdate(prev); err != nil {
			return errors.Wrap(err, fmt.Sprintf("pod: %s", prev.GetName()))
		}
	}

	if next != nil {
		if err := beforeUpdate(next); err != nil {
			return errors.Wrap(err, fmt.Sprintf("pod: %s", next.GetName()))
		}
	}

	return nil
}

func beforeUpdate(pod *v1.Pod) error {
	host := pod.Status.PodIP

	if err := isGreen(host); err != nil {
		return errors.Wrap(err, "es cluster not yet green")
	}

	if err := setAllocation(host, "new_primaries"); err != nil {
		return errors.Wrap(err, "failed to set allocation to new_primaries")
	}

	if err := flushSync(host); err != nil {
		return errors.Wrap(err, "flush sync failed for es pod")
	}

	return nil
}

func afterUpdate(pod *v1.Pod) error {
	host := pod.Status.PodIP

	if err := setAllocation(host, "all"); err != nil {
		return errors.Wrap(err, "failed to set allocation to all")
	}

	if err := isGreen(host); err != nil {
		return errors.Wrap(err, "es cluster not yet green")
	}

	return nil
}

package factory

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/bpineau/statefulset-pilot/pkg/hooks"
	"github.com/bpineau/statefulset-pilot/pkg/hooks/elasticsearch"
	"github.com/bpineau/statefulset-pilot/pkg/hooks/noop"
)

type HookFactory func() (hooks.STSRolloutHooks, error)

var registry = make(map[string]HookFactory)

func Register(name string, factory HookFactory) {
	registry[name] = factory
}

func Get(sts *appsv1.StatefulSet, key string) (hooks.STSRolloutHooks, error) {
	label, ok := sts.GetLabels()[key]
	if !ok {
		return nil, fmt.Errorf("missing %s label", key)
	}

	h, ok := registry[label]
	if !ok {
		return nil, fmt.Errorf("unsupported hook manager: %s", label)
	}

	return h()
}

func init() {
	Register("elasticsearch", elasticsearch.New)
	Register("noop", noop.New)
}
